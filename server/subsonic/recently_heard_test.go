package subsonic

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetRecentlyHeard", func() {
	var router *Router
	var ctx context.Context
	at := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

	BeforeEach(func() {
		ds := &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		ctx = request.WithUser(context.Background(), model.User{ID: "u1", UserName: "u1"})
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).Heard = []model.HeardSong{
			{ID: "new", Heard: at.Add(time.Hour)},
			{ID: "old", Heard: at.Add(-time.Hour)},
		}
	})

	// RD: since is read in seconds, or ignored, so Smart Shuffle is told of songs heard long before the cooldown,
	// or of none.
	It("lists the songs heard at since (milliseconds) or later", func() {
		r := newGetRequest(fmt.Sprintf("since=%d", at.UnixMilli())).WithContext(ctx)

		resp, err := router.GetRecentlyHeard(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.RecentlyHeard.Song).To(HaveLen(1))
		Expect(resp.RecentlyHeard.Song[0].ID).To(Equal("new"))
		Expect(resp.RecentlyHeard.Song[0].Heard).To(Equal(at.Add(time.Hour)))
	})

	// RD: a missing since is taken as the epoch, sending the user's whole history on every Smart Shuffle refill.
	It("requires since", func() {
		_, err := router.GetRecentlyHeard(newGetRequest().WithContext(ctx))

		Expect(err).To(HaveOccurred())
	})
})
