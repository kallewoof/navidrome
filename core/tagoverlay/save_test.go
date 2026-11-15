package tagoverlay_test

import (
	"context"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/tagoverlay"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeScanner records the folders it was asked to scan; it reports busy for its first busyFor calls.
type fakeScanner struct {
	model.Scanner
	mu      sync.Mutex
	busyFor int
	calls   [][]model.ScanTarget
}

func (f *fakeScanner) ScanFolders(_ context.Context, fullScan bool, targets []model.ScanTarget) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if fullScan {
		// A full scan re-reads every file under the folder; recorded so the comparison in the test fails.
		targets = append([]model.ScanTarget{{FolderPath: "FULL SCAN"}}, targets...)
	}
	f.calls = append(f.calls, targets)
	if len(f.calls) <= f.busyFor {
		return nil, scanner.ErrAlreadyScanning
	}
	return nil, nil
}

func (f *fakeScanner) scans() [][]model.ScanTarget {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]model.ScanTarget(nil), f.calls...)
}

var _ = Describe("Save and Reset", func() {
	var (
		ds       *tests.MockDataStore
		overlays *tests.MockTagOverlayRepo
		sc       *fakeScanner
		ctx      context.Context
	)
	folder := []model.ScanTarget{{LibraryID: 1, FolderPath: "Unknown Artist/Unknown Album"}}

	BeforeEach(func() {
		ctx = context.Background()
		overlays = tests.CreateMockTagOverlayRepo()
		files := tests.CreateMockMediaFileRepo()
		files.SetData(model.MediaFiles{{ID: "s1", LibraryID: 1, Path: "Unknown Artist/Unknown Album/song.mp3"}})
		ds = &tests.MockDataStore{MockedTagOverlay: overlays, MockedMediaFile: files}
		sc = &fakeScanner{}
		DeferCleanup(func(retry time.Duration) { tagoverlay.RescanRetry = retry }, tagoverlay.RescanRetry)
		tagoverlay.RescanRetry = time.Millisecond
	})

	// RD: Save stores the edit but does not mark the song or start a scan of its folder, so the edit shows only
	// after a full scan, or Save replaces the song's earlier edit instead of adding to it.
	It("adds the edit to the song's overlay and re-reads the song's folder", func() {
		overlays.Data["s1"] = model.TagOverlay{MediaFileID: "s1", Tags: map[string][]string{"title": {"Song 1"}}}

		_, err := tagoverlay.Save(ctx, ds, sc, "s1", "user", tagoverlay.Fields{Composer: ptr("Composer A")})

		Expect(err).ToNot(HaveOccurred())
		Expect(overlays.Data["s1"].Tags).To(Equal(map[string][]string{"title": {"Song 1"}, "composer": {"Composer A"}}))
		Expect(overlays.Data["s1"].UpdatedBy).To(Equal("user"))
		Expect(overlays.Marked).To(Equal([]string{"s1"}))
		Eventually(sc.scans).Should(Equal([][]model.ScanTarget{folder}))
	})

	// RD: a scanner busy with another scan makes the re-read give up at once, so the edit waits for the next
	// scheduled scan, hours later.
	It("tries again while another scan holds the scanner", func() {
		sc.busyFor = 2

		_, err := tagoverlay.Save(ctx, ds, sc, "s1", "user", tagoverlay.Fields{Title: ptr("Song 1")})

		Expect(err).ToNot(HaveOccurred())
		Eventually(sc.scans).Should(HaveLen(3))
	})

	// RD: an edit for an ID that is not a song (or one the user cannot see) is stored anyway, and stays in the
	// overlay list forever with no file to apply it to.
	It("stores nothing for a song that does not exist", func() {
		_, err := tagoverlay.Save(ctx, ds, sc, "nope", "user", tagoverlay.Fields{Title: ptr("Song 1")})

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(overlays.Data).To(BeEmpty())
		Consistently(sc.scans, "50ms").Should(BeEmpty())
	})

	// RD: Reset deletes the overlay but leaves the song unmarked, so the library keeps the edited tags.
	It("drops the overlay and re-reads the song's folder on reset", func() {
		overlays.Data["s1"] = model.TagOverlay{MediaFileID: "s1", Tags: map[string][]string{"title": {"Song 1"}}}

		Expect(tagoverlay.Reset(ctx, ds, sc, "s1")).To(Succeed())

		Expect(overlays.Data).To(BeEmpty())
		Expect(overlays.Marked).To(Equal([]string{"s1"}))
		Eventually(sc.scans).Should(Equal([][]model.ScanTarget{folder}))
	})
})
