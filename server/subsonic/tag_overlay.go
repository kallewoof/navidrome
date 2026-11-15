package subsonic

import (
	"errors"
	"net/http"
	"slices"
	"sort"

	"github.com/navidrome/navidrome/core/tagoverlay"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// GetTagOverlay returns the song's tag edit; a song without one has only its id.
func (api *Router) GetTagOverlay(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := req.Params(r).String("id")
	if err != nil {
		return nil, err
	}
	if _, err := api.ds.MediaFile(ctx).Get(id); err != nil {
		return nil, songError(err)
	}
	o, err := api.ds.TagOverlay(ctx).Get(id)
	if errors.Is(err, model.ErrNotFound) {
		o = &model.TagOverlay{MediaFileID: id}
	} else if err != nil {
		return nil, err
	}
	response := newResponse()
	response.TagOverlay = overlayResponse(o)
	return response, nil
}

// SetTagOverlay applies an edit to the song's tags: each of title, artist, composer and album that is present
// replaces that field (empty clears it), and the song is re-read with it.
func (api *Router) SetTagOverlay(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	f := tagoverlay.Fields{Title: p.StringPtr("title"), Artist: p.StringPtr("artist"),
		Composer: p.StringPtr("composer"), Album: p.StringPtr("album")}
	if f.Title == nil && f.Artist == nil && f.Composer == nil && f.Album == nil {
		return nil, newError(responses.ErrorMissingParameter, "one of title, artist, composer or album is required")
	}
	user, _ := request.UserFrom(ctx)
	o, err := tagoverlay.Save(ctx, api.ds, api.scanner, id, user.UserName, f)
	if errors.Is(err, tagoverlay.ErrEmptyTitle) {
		return nil, newError(responses.ErrorGeneric, err.Error())
	}
	if err != nil {
		return nil, songError(err)
	}
	log.Info(ctx, "Tag overlay saved", "id", id, "user", user.UserName)
	response := newResponse()
	response.TagOverlay = overlayResponse(o)
	return response, nil
}

// ResetTagOverlay drops the song's edit, so it shows its file's own tags again.
func (api *Router) ResetTagOverlay(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	id, err := req.Params(r).String("id")
	if err != nil {
		return nil, err
	}
	if err := tagoverlay.Reset(ctx, api.ds, api.scanner, id); err != nil {
		return nil, songError(err)
	}
	return newResponse(), nil
}

// GetTagOverlays lists every overlay as raw tags with its file's path, for writing them into the files.
func (api *Router) GetTagOverlays(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	all, err := api.ds.TagOverlay(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	list := &responses.TagOverlays{Overlay: []responses.TagOverlayFile{}}
	for _, o := range all {
		mf, err := api.ds.MediaFile(ctx).Get(o.MediaFileID)
		if errors.Is(err, model.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(o.Tags))
		for name := range o.Tags {
			names = append(names, name)
		}
		sort.Strings(names)
		file := responses.TagOverlayFile{ID: o.MediaFileID, Path: mf.Path, Changed: o.UpdatedAt, ChangedBy: o.UpdatedBy}
		for _, name := range names {
			file.Tag = append(file.Tag, responses.OverlayTag{Name: name, Value: slices.Clone(o.Tags[name])})
		}
		list.Overlay = append(list.Overlay, file)
	}
	response := newResponse()
	response.TagOverlays = list
	return response, nil
}

func overlayResponse(o *model.TagOverlay) *responses.TagOverlay {
	resp := &responses.TagOverlay{ID: o.MediaFileID, ChangedBy: o.UpdatedBy}
	if !o.UpdatedAt.IsZero() {
		resp.Changed = &o.UpdatedAt
	}
	for field, value := range tagoverlay.Edited(o.Tags) {
		v := value
		switch field {
		case "title":
			resp.Title = &v
		case "artist":
			resp.Artist = &v
		case "composer":
			resp.Composer = &v
		case "album":
			resp.Album = &v
		}
	}
	return resp
}

func songError(err error) error {
	if errors.Is(err, model.ErrNotFound) {
		return newError(responses.ErrorDataNotFound, "Song not found")
	}
	return err
}
