package metadata_test

import (
	"os"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplyOverlay", func() {
	var props metadata.Info

	BeforeEach(func() {
		_, filePath, _ := tests.TempFile(GinkgoT(), "test", ".mp3")
		fileInfo, _ := os.Stat(filePath)
		props = metadata.Info{FileInfo: testFileInfo{fileInfo}}
	})

	toMediaFile := func(raw model.RawTags, overlay map[string][]string) model.MediaFile {
		props.Tags = metadata.ApplyOverlay(raw, overlay)
		return metadata.New("filepath", props).ToMediaFile(1, "folderID")
	}
	names := func(mf model.MediaFile, role model.Role) []string {
		var out []string
		for _, p := range mf.Participants[role] {
			out = append(out, p.Name)
		}
		return out
	}

	// RD: the overlay's artist is added next to the file's own frame (TPE1) instead of replacing it, and the
	// mapping merges the two, so the wrong artist stays credited beside the correction.
	It("replaces the file's value under any alias, and keeps the tags it does not set", func() {
		mf := toMediaFile(model.RawTags{"tpe1": {"Wrong Artist"}, "title": {"Song 1"}},
			map[string][]string{"artist": {"Artist A"}})

		Expect(mf.Artist).To(Equal("Artist A"))
		Expect(names(mf, model.RoleArtist)).To(Equal([]string{"Artist A"}))
		Expect(mf.Title).To(Equal("Song 1"))
	})

	// RD: an empty overlay value is skipped instead of clearing the tag, so the file's old ARTISTS keeps deciding
	// the artists after the credit was corrected to a single artist.
	It("removes a tag the overlay sets to no values", func() {
		mf := toMediaFile(model.RawTags{"artist": {"Old A feat. Old B"}, "artists": {"Old A", "Old B"}},
			map[string][]string{"artist": {"Artist A"}, "artists": {}})

		Expect(names(mf, model.RoleArtist)).To(Equal([]string{"Artist A"}))
	})

	// RD: only the overlay's own performer:<role> key is replaced, so the file's credits under other roles stay
	// listed next to the corrected voice.
	It("replaces every performer credit of the file when the overlay sets one", func() {
		mf := toMediaFile(model.RawTags{"performer:guitar": {"Old Player"}, "performer": {"Old Singer (vocals)"}},
			map[string][]string{"performer:append (dark)": {"Voice A"}})

		performers := mf.Participants[model.RolePerformer]
		Expect(performers).To(HaveLen(1))
		Expect(performers[0].Name).To(Equal("Voice A"))
		Expect(performers[0].SubRole).To(Equal("Append (Dark)"))
	})
})
