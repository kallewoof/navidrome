package tagoverlay_test

import (
	"github.com/navidrome/navidrome/core/tagoverlay"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ptr(s string) *string { return &s }

var _ = Describe("Merge", func() {
	edit := func(existing map[string][]string, f tagoverlay.Fields) map[string][]string {
		tags, err := tagoverlay.Merge(existing, f)
		Expect(err).ToNot(HaveOccurred())
		return tags
	}

	// RD: a featured voice keeps its voice bank in the artist list, so each bank becomes an artist of its own
	// instead of one page for the singer, and the bank is lost as a performer credit.
	It("splits a credit into producers and plain voices, and keeps each bank as the voice's performer role", func() {
		tags := edit(nil, tagoverlay.Fields{Artist: ptr("Producer A feat. Voice A Append (Dark), Voice A Append (Vivid)")})

		Expect(tags).To(Equal(map[string][]string{
			"artist":                   {"Producer A feat. Voice A Append (Dark), Voice A Append (Vivid)"},
			"artists":                  {"Producer A", "Voice A"},
			"performer:append (dark)":  {"Voice A"},
			"performer:append (vivid)": {"Voice A"},
		}))
	})

	// RD: banks written before the name ("V3 GUMI") are read as part of it, and VocaDB's "(Unknown)" is kept as a
	// bank, so the singer gets a second artist page and a performer role named "Unknown".
	It("reads leading bank words, and drops a bank VocaDB marks unknown", func() {
		tags := edit(nil, tagoverlay.Fields{Artist: ptr("Producer A ft. V3 Voice B (Sweet), Voice C Append (Unknown), Voice D")})

		Expect(tags["artist"]).To(Equal([]string{"Producer A ft. V3 Voice B (Sweet), Voice C Append, Voice D"}))
		Expect(tags["artists"]).To(Equal([]string{"Producer A", "Voice B", "Voice C", "Voice D"}))
		Expect(tags["performer:v3 (sweet)"]).To(Equal([]string{"Voice B"}))
		Expect(tags["performer:append"]).To(Equal([]string{"Voice C"}))
		Expect(tags["performer:vocals"]).To(Equal([]string{"Voice D"}))
	})

	// RD: a credit without "feat." leaves the file's ARTISTS and performer credits alone, and the scanner, which
	// prefers ARTISTS, keeps showing the old artists under the corrected credit.
	It("clears the artist list and performers for a credit without featured voices", func() {
		tags := edit(map[string][]string{"artist": {"Old"}, "artists": {"Old", "Voice A"}, "performer:append (dark)": {"Voice A"}},
			tagoverlay.Fields{Artist: ptr("Artist A")})

		Expect(tags).To(Equal(map[string][]string{"artist": {"Artist A"}, "artists": {}, "performer": {}}))
	})

	// RD: an edit replaces the whole overlay instead of the fields it carries, so fixing the title undoes an earlier
	// artist fix; or an empty composer is skipped instead of clearing a wrong one.
	It("changes only the fields the edit carries, and an empty one clears its tag", func() {
		tags := edit(map[string][]string{"artist": {"Artist A"}, "composer": {"Wrong Composer"}},
			tagoverlay.Fields{Title: ptr(" Song 1 "), Composer: ptr("")})

		Expect(tags).To(Equal(map[string][]string{"artist": {"Artist A"}, "title": {"Song 1"}, "composer": {}}))
	})

	// RD: an empty title is stored, and the scanner falls back to the file name, so a slip of the keyboard renames the
	// song after its file.
	It("refuses an empty title", func() {
		_, err := tagoverlay.Merge(nil, tagoverlay.Fields{Title: ptr("  ")})

		Expect(err).To(MatchError(tagoverlay.ErrEmptyTitle))
	})
})
