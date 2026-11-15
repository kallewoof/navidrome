package subsonic

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type idleScanner struct{ model.Scanner }

func (idleScanner) ScanFolders(context.Context, bool, []model.ScanTarget) ([]string, error) {
	return nil, nil
}

var _ = Describe("TagOverlay", func() {
	var (
		router   *Router
		overlays *tests.MockTagOverlayRepo
	)

	BeforeEach(func() {
		overlays = tests.CreateMockTagOverlayRepo()
		files := tests.CreateMockMediaFileRepo()
		files.SetData(model.MediaFiles{{ID: "s1", LibraryID: 1, Path: "Artist A/Album 1/song1.mp3"}})
		ds := &tests.MockDataStore{MockedTagOverlay: overlays, MockedMediaFile: files}
		router = New(ds, nil, nil, nil, nil, nil, idleScanner{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	code := func(err error) int32 {
		var se subError
		Expect(errors.As(err, &se)).To(BeTrue(), "a Subsonic error, got %v", err)
		return se.code
	}

	// RD: absent fields are read as empty ones, so fixing a title also clears the song's artist and album; or an
	// empty field is ignored, so a wrong composer can never be cleared.
	It("changes the fields a request carries, clears empty ones, and leaves absent ones", func() {
		resp, err := router.SetTagOverlay(newGetRequest("id=s1", "title=Song%201", "composer="))

		Expect(err).ToNot(HaveOccurred())
		Expect(overlays.Data["s1"].Tags).To(Equal(map[string][]string{"title": {"Song 1"}, "composer": {}}))
		Expect(resp.TagOverlay.Title).To(HaveValue(Equal("Song 1")))
		Expect(resp.TagOverlay.Composer).To(HaveValue(BeEmpty()))
		Expect(resp.TagOverlay.Artist).To(BeNil())
	})

	// RD: reading a song without an edit fails as not found, so the edit sheet cannot open for an unedited song;
	// or an edit's cleared field reads back as absent, so the sheet shows the file's value as if never changed.
	It("reads back a song's edit, and only the id for a song without one", func() {
		resp, err := router.GetTagOverlay(newGetRequest("id=s1"))
		Expect(err).ToNot(HaveOccurred())
		Expect(*resp.TagOverlay).To(Equal(responses.TagOverlay{ID: "s1"}))

		overlays.Data["s1"] = model.TagOverlay{MediaFileID: "s1", Tags: map[string][]string{
			"artist": {"Artist A"}, "artists": {}, "performer": {}, "composer": {}}}
		resp, err = router.GetTagOverlay(newGetRequest("id=s1"))

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.TagOverlay.Artist).To(HaveValue(Equal("Artist A")))
		Expect(resp.TagOverlay.Composer).To(HaveValue(BeEmpty()))
		Expect(resp.TagOverlay.Title).To(BeNil())
	})

	// RD: a request with no field to change is accepted and stores an empty overlay, which then shows as an edit.
	It("refuses a request that changes nothing", func() {
		_, err := router.SetTagOverlay(newGetRequest("id=s1"))

		Expect(code(err)).To(Equal(responses.ErrorMissingParameter))
		Expect(overlays.Data).To(BeEmpty())
	})

	// RD: an edit for a song that is not there comes back as a generic failure, which Musly shows as "try again".
	It("answers data not found for a song that is not there", func() {
		_, err := router.SetTagOverlay(newGetRequest("id=none", "title=Song%201"))

		Expect(code(err)).To(Equal(responses.ErrorDataNotFound))
	})

	// RD: the list for the batch apply includes overlays of songs no longer in the library, or tags in random order,
	// so the apply step chokes on a missing path or the list changes between two reads of the same overlays.
	It("lists each song's raw tags in name order with its file, skipping songs that are gone", func() {
		overlays.Data["s1"] = model.TagOverlay{MediaFileID: "s1", Tags: map[string][]string{
			"title": {"Song 1"}, "artists": {}, "artist": {"Artist A"}}}
		overlays.Data["gone"] = model.TagOverlay{MediaFileID: "gone", Tags: map[string][]string{"title": {"X"}}}

		resp, err := router.GetTagOverlays(newGetRequest())

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.TagOverlays.Overlay).To(HaveLen(1))
		file := resp.TagOverlays.Overlay[0]
		Expect(file.Path).To(Equal("Artist A/Album 1/song1.mp3"))
		Expect(file.Tag).To(Equal([]responses.OverlayTag{
			{Name: "artist", Value: []string{"Artist A"}}, {Name: "artists", Value: []string{}}, {Name: "title", Value: []string{"Song 1"}}}))
	})
})
