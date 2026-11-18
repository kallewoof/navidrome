package subsonic

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// GetRecentlyHeard lists the songs the user played or skipped, on any device, at `since` (milliseconds since the
// epoch) or later, newest first (extension unimouseRecentlyHeard). Smart Shuffle keeps them out of its picks.
func (api *Router) GetRecentlyHeard(r *http.Request) (*responses.Subsonic, error) {
	since, err := req.Params(r).Int64("since")
	if err != nil {
		return nil, err
	}
	heard, err := api.ds.MediaFile(r.Context()).RecentlyHeard(time.UnixMilli(since))
	if err != nil {
		return nil, err
	}
	songs := make([]responses.HeardSong, len(heard))
	for i, h := range heard {
		songs[i] = responses.HeardSong{ID: h.ID, Heard: h.Heard}
	}
	response := newResponse()
	response.RecentlyHeard = &responses.RecentlyHeard{Song: songs}
	return response, nil
}
