package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagOverlayRepository", func() {
	var repo model.TagOverlayRepository

	BeforeEach(func() {
		repo = NewTagOverlayRepository(log.NewContext(context.TODO()), GetDBXBuilder())
		DeferCleanup(func() {
			for _, id := range []string{"o1", "o2", "o3"} {
				Expect(repo.Delete(id)).To(Succeed())
			}
		})
	})

	// RD: a second edit of a song fails on the primary key, or is stored next to the first, so only the first edit
	// ever applies; or the empty list that clears a tag comes back without its key and stops clearing it.
	It("replaces a song's overlay on a second save, keeping empty lists", func() {
		Expect(repo.Put(&model.TagOverlay{MediaFileID: "o1", Tags: map[string][]string{"title": {"Song 1"}}, UpdatedBy: "a"})).To(Succeed())
		Expect(repo.Put(&model.TagOverlay{MediaFileID: "o1", Tags: map[string][]string{"artist": {"Artist A"}, "artists": {}}, UpdatedBy: "b"})).To(Succeed())

		o, err := repo.Get("o1")
		Expect(err).ToNot(HaveOccurred())
		Expect(o.Tags).To(Equal(map[string][]string{"artist": {"Artist A"}, "artists": {}}))
		Expect(o.UpdatedBy).To(Equal("b"))
		all, err := repo.GetAll()
		Expect(err).ToNot(HaveOccurred())
		Expect(all).To(HaveLen(1))
	})

	// RD: the scanner's lookup returns overlays of songs it did not ask for, or misses some it did, applying edits
	// to the wrong songs or not at all.
	It("returns the overlays of exactly the asked songs that have one", func() {
		for _, id := range []string{"o1", "o2", "o3"} {
			Expect(repo.Put(&model.TagOverlay{MediaFileID: id, Tags: map[string][]string{"title": {id}}})).To(Succeed())
		}

		got, err := repo.GetMany("o1", "o3", "none")

		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(2))
		Expect(got["o3"].Tags).To(Equal(map[string][]string{"title": {"o3"}}))
		Expect(got).To(HaveKey("o1"))
	})

	// RD: a song without an overlay reads as a generic error instead of not found, so the API fails for every
	// unedited song instead of answering that it has no edit; or Delete leaves the row, so a reset keeps the edit.
	It("reads a song without an overlay as not found, also after its overlay is deleted", func() {
		_, err := repo.Get("o1")
		Expect(err).To(MatchError(model.ErrNotFound))

		Expect(repo.Put(&model.TagOverlay{MediaFileID: "o1", Tags: map[string][]string{"title": {"Song 1"}}})).To(Succeed())
		Expect(repo.Delete("o1")).To(Succeed())

		_, err = repo.Get("o1")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	// RD: a lookup with no IDs, which the scanner makes for a folder of new files, fails or returns every overlay.
	It("returns no overlays for no IDs", func() {
		Expect(repo.Put(&model.TagOverlay{MediaFileID: "o1", Tags: map[string][]string{"title": {"Song 1"}}})).To(Succeed())

		got, err := repo.GetMany()

		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeEmpty())
	})
})
