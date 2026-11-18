package subsonic

import (
	"context"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmarks", func() {
	var router *Router
	var ds *tests.MockDataStore
	var mfRepo *tests.MockMediaFileRepo
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		ctx = request.WithUser(context.Background(), model.User{ID: "u1", UserName: "u1"})
		mfRepo = ds.MediaFile(ctx).(*tests.MockMediaFileRepo)
		mfRepo.SetData(model.MediaFiles{{ID: "visible"}})
	})

	Describe("CreateBookmark", func() {
		It("rejects an id the user cannot read", func() {
			r := newGetRequest("id=hidden", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).To(HaveOccurred())
			Expect(mapToSubsonicError(err).code).To(Equal(responses.ErrorDataNotFound))
			Expect(mfRepo.BookmarksAdded).To(BeEmpty())
		})

		It("accepts an id the user can read", func() {
			r := newGetRequest("id=visible", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(mfRepo.BookmarksAdded).To(ConsistOf("visible"))
		})
	})

	Describe("SavePlayQueueByIndex", func() {
		var pqRepo *tests.MockPlayQueueRepo

		BeforeEach(func() {
			pqRepo = ds.PlayQueue(ctx).(*tests.MockPlayQueueRepo)
		})

		// RD: the state is dropped on save or not returned, so a device that adopts the queue plays it without the
		// saving device's Smart Shuffle.
		It("returns the client's state with the queue it was saved with", func() {
			state := `{"refill":"smartShuffle"}`
			_, err := router.SavePlayQueueByIndex(newGetRequest("id=visible", "currentIndex=0",
				"unimouseState="+url.QueryEscape(state)).WithContext(ctx))
			Expect(err).ToNot(HaveOccurred())

			resp, err := router.GetPlayQueueByIndex(newGetRequest().WithContext(ctx))

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PlayQueueByIndex.UnimouseState).To(Equal(state))
		})

		// RD: any string is stored, so a client bug saves state no device can read, and every adopting device fails
		// on it; or an unbounded one fills the database.
		DescribeTable("refuses state that is not a JSON object, or too large, and saves nothing",
			func(state string) {
				_, err := router.SavePlayQueueByIndex(newGetRequest("id=visible", "currentIndex=0",
					"unimouseState="+url.QueryEscape(state)).WithContext(ctx))

				Expect(err).To(HaveOccurred())
				Expect(pqRepo.Queue).To(BeNil())
			},
			Entry("not JSON", "not json"),
			Entry("a JSON array", "[1,2]"),
			Entry("over 64 KiB", `{"x":"`+strings.Repeat("a", 64<<10)+`"}`),
		)
	})
})
