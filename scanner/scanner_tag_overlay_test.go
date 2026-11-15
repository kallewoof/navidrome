package scanner_test

import (
	"context"
	"path/filepath"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tag overlays", Ordered, func() {
	var ctx context.Context
	var ds model.DataStore
	var s model.Scanner
	var lib model.Library

	BeforeAll(func() {
		tests.SkipOnWindows("SQLite file lock blocks TempDir cleanup (#TBD-path-sep-scanner)")
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "123", IsAdmin: true})
		conf.Server.DbPath = filepath.Join(GinkgoT().TempDir(), "test-tag-overlay.db?_journal_mode=WAL")
		db.Db().SetMaxOpenConns(1)
	})

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.MusicFolder = "fake:///music"
		conf.Server.DevExternalScanner = false
		db.Init(ctx)
		DeferCleanup(func() { Expect(tests.ClearDB()).To(Succeed()) })
		ds = persistence.New(db.Db())
		Expect(ds.User(ctx).Put(&model.User{ID: "123", UserName: "admin", IsAdmin: true, NewPassword: "p"})).To(Succeed())
		s = scanner.New(ctx, ds, events.NoopBroker(), playlists.NewPlaylists(ds, artwork.NewUploader(ds)), metrics.NewNoopInstance())
		lib = model.Library{ID: 1, Name: "Fake Library", Path: "fake:///music"}
		Expect(ds.Library(ctx).Put(&lib)).To(Succeed())

		untagged := template(_t{"artist": "Wrong Artist", "album": "Album 1"})
		_ = createFS(fstest.MapFS{
			"unknown/song1.mp3": untagged(track(1, "Song 1")),
			"unknown/song2.mp3": untagged(track(2, "Song 2")),
		})
		_, err := s.ScanAll(ctx, false)
		Expect(err).ToNot(HaveOccurred())
	})

	song := func(path string) model.MediaFile {
		mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"media_file.path": path}})
		Expect(err).ToNot(HaveOccurred())
		Expect(mfs).To(HaveLen(1))
		return mfs[0]
	}
	quickScan := func() {
		_, err := s.ScanFolders(ctx, false, []model.ScanTarget{{LibraryID: lib.ID, FolderPath: "unknown"}})
		Expect(err).ToNot(HaveOccurred())
	}

	// RD: the scanner skips overlays, or a quick scan never re-reads the unchanged file, so an edit saved from an app
	// never reaches the library (or replaces the song's ID, losing its plays and stars).
	It("re-reads a marked song with its overlay on the next quick scan of its folder", func() {
		before := song("unknown/song1.mp3")
		Expect(ds.TagOverlay(ctx).Put(&model.TagOverlay{MediaFileID: before.ID, Tags: map[string][]string{
			"artist": {"Artist A"}, "title": {"Song One"}}})).To(Succeed())
		Expect(ds.TagOverlay(ctx).MarkForRescan(before.ID)).To(Succeed())

		quickScan()

		after := song("unknown/song1.mp3")
		Expect(after.ID).To(Equal(before.ID))
		Expect([]string{after.Title, after.Artist}).To(Equal([]string{"Song One", "Artist A"}))
		Expect(song("unknown/song2.mp3").Artist).To(Equal("Wrong Artist"))
	})

	// RD: a reset deletes the overlay but the unchanged file is not re-read, so the library keeps the edited tags
	// although the overlay is gone.
	It("goes back to the file's own tags once the overlay is deleted and the song re-read", func() {
		id := song("unknown/song1.mp3").ID
		Expect(ds.TagOverlay(ctx).Put(&model.TagOverlay{MediaFileID: id, Tags: map[string][]string{
			"artist": {"Artist A"}}})).To(Succeed())
		Expect(ds.TagOverlay(ctx).MarkForRescan(id)).To(Succeed())
		quickScan()
		Expect(song("unknown/song1.mp3").Artist).To(Equal("Artist A"))

		Expect(ds.TagOverlay(ctx).Delete(id)).To(Succeed())
		Expect(ds.TagOverlay(ctx).MarkForRescan(id)).To(Succeed())
		quickScan()

		Expect(song("unknown/song1.mp3").Artist).To(Equal("Wrong Artist"))
	})
})
