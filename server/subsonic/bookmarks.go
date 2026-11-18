package subsonic

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) GetBookmarks(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.MediaFile(r.Context())
	bookmarks, err := repo.GetBookmarks()
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Bookmarks = &responses.Bookmarks{}
	response.Bookmarks.Bookmark = slice.Map(bookmarks, func(bmk model.Bookmark) responses.Bookmark {
		return responses.Bookmark{
			Entry:    childFromMediaFile(r.Context(), bmk.Item),
			Position: bmk.Position,
			Username: user.UserName,
			Comment:  bmk.Comment,
			Created:  bmk.CreatedAt,
			Changed:  bmk.UpdatedAt,
		}
	})
	return response, nil
}

func (api *Router) CreateBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	comment, _ := p.String("comment")
	position := p.Int64Or("position", 0)

	repo := api.ds.MediaFile(r.Context())
	ok, err := repo.Exists(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, newError(responses.ErrorDataNotFound, "Song not found")
	}

	err = repo.AddBookmark(id, comment, position)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeleteBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	repo := api.ds.MediaFile(r.Context())
	err = repo.DeleteBookmark(id)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetPlayQueue(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.PlayQueue(r.Context())
	pq, err := repo.RetrieveWithMediaFiles(user.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if pq == nil || len(pq.Items) == 0 {
		response := newResponse()
		response.PlayQueue = &responses.PlayQueue{
			Username: user.UserName,
		}
		return response, nil
	}

	response := newResponse()
	var currentID string
	if pq.Current >= 0 && pq.Current < len(pq.Items) {
		currentID = pq.Items[pq.Current].ID
	}
	response.PlayQueue = &responses.PlayQueue{
		Entry:     slice.MapWithArg(pq.Items, r.Context(), childFromMediaFile),
		Current:   currentID,
		Position:  pq.Position,
		Username:  user.UserName,
		Changed:   pq.UpdatedAt,
		ChangedBy: pq.ChangedBy,
	}
	return response, nil
}

func (api *Router) SavePlayQueue(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids := p.Strings("id")
	currentID, _ := p.String("current")
	position := p.Int64Or("position", 0)

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	items := slice.Map(ids, func(id string) model.MediaFile {
		return model.MediaFile{ID: id}
	})

	currentIndex := 0
	for i, id := range ids {
		if id == currentID {
			currentIndex = i
			break
		}
	}

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   currentIndex,
		Position:  position,
		ChangedBy: client,
		Items:     items,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	repo := api.ds.PlayQueue(r.Context())
	err := repo.Store(pq)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetPlayQueueByIndex(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.PlayQueue(r.Context())
	pq, err := repo.RetrieveWithMediaFiles(user.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if pq == nil || len(pq.Items) == 0 {
		response := newResponse()
		response.PlayQueueByIndex = &responses.PlayQueueByIndex{
			Username: user.UserName,
		}
		return response, nil
	}

	response := newResponse()

	var index *int
	if len(pq.Items) > 0 {
		index = &pq.Current
	}

	response.PlayQueueByIndex = &responses.PlayQueueByIndex{
		Entry:         slice.MapWithArg(pq.Items, r.Context(), childFromMediaFile),
		CurrentIndex:  index,
		Position:      pq.Position,
		Username:      user.UserName,
		Changed:       pq.UpdatedAt,
		ChangedBy:     pq.ChangedBy,
		UnimouseState: pq.ClientState,
	}
	return response, nil
}

// maxQueueStateBytes bounds the client state saved with a queue: room for the reasons of a few hundred songs.
const maxQueueStateBytes = 64 << 10

// queueState reads the optional unimouseState parameter: a JSON object, stored as sent. A client that does not
// send one saves a queue without state, which clears the state of the queue it replaces.
func queueState(p *req.Values) (string, error) {
	state, _ := p.String("unimouseState")
	if state == "" {
		return "", nil
	}
	if len(state) > maxQueueStateBytes {
		return "", newError(responses.ErrorGeneric, "unimouseState is %d bytes, over the limit of %d", len(state), maxQueueStateBytes)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(state), &obj); err != nil {
		return "", newError(responses.ErrorGeneric, "unimouseState is not a JSON object: %s", err)
	}
	return state, nil
}

func (api *Router) SavePlayQueueByIndex(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids := p.Strings("id")
	state, err := queueState(p)
	if err != nil {
		return nil, err
	}

	position := p.Int64Or("position", 0)

	var currentIndex int

	if len(ids) > 0 {
		currentIndex, err = p.Int("currentIndex")
		if err != nil || currentIndex < 0 || currentIndex >= len(ids) {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}
	}

	items := slice.Map(ids, func(id string) model.MediaFile {
		return model.MediaFile{ID: id}
	})

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	pq := &model.PlayQueue{
		UserID:      user.ID,
		Current:     currentIndex,
		Position:    position,
		ChangedBy:   client,
		ClientState: state,
		Items:       items,
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}

	repo := api.ds.PlayQueue(r.Context())
	err = repo.Store(pq)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}
