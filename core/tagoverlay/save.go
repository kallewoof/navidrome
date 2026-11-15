package tagoverlay

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
)

// RescanRetry is how long Save waits between tries while another scan holds the scanner, and RescanTries how often
// it tries. The mark stays in the database, so a song a busy scanner never reached is re-read by the next scan.
var (
	RescanRetry = 5 * time.Second
	RescanTries = 24
)

// Save applies the edit to the song's overlay and has the scanner re-read the song with it, in the background.
func Save(ctx context.Context, ds model.DataStore, sc model.Scanner, mediaFileID, user string, f Fields) (*model.TagOverlay, error) {
	mf, err := ds.MediaFile(ctx).Get(mediaFileID)
	if err != nil {
		return nil, err
	}
	repo := ds.TagOverlay(ctx)
	existing := map[string][]string{}
	if o, err := repo.Get(mediaFileID); err == nil {
		existing = o.Tags
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	tags, err := Merge(existing, f)
	if err != nil {
		return nil, err
	}
	o := &model.TagOverlay{MediaFileID: mediaFileID, Tags: tags, UpdatedBy: user}
	if err := repo.Put(o); err != nil {
		return nil, err
	}
	if err := rescan(ctx, ds, sc, mf); err != nil {
		return nil, err
	}
	return o, nil
}

// Reset drops the song's overlay, so the scanner re-reads it with its file's own tags.
func Reset(ctx context.Context, ds model.DataStore, sc model.Scanner, mediaFileID string) error {
	mf, err := ds.MediaFile(ctx).Get(mediaFileID)
	if err != nil {
		return err
	}
	if err := ds.TagOverlay(ctx).Delete(mediaFileID); err != nil {
		return err
	}
	return rescan(ctx, ds, sc, mf)
}

func rescan(ctx context.Context, ds model.DataStore, sc model.Scanner, mf *model.MediaFile) error {
	if err := ds.TagOverlay(ctx).MarkForRescan(mf.ID); err != nil {
		return err
	}
	target := []model.ScanTarget{{LibraryID: mf.LibraryID, FolderPath: path.Dir(mf.Path)}}
	bg := context.WithoutCancel(ctx)
	go func() {
		for try := 1; try <= RescanTries; try++ {
			_, err := sc.ScanFolders(bg, false, target)
			if !errors.Is(err, scanner.ErrAlreadyScanning) {
				if err != nil {
					log.Error(bg, "Rescan for a tag overlay failed", "file", mf.Path, err)
				}
				return
			}
			time.Sleep(RescanRetry)
		}
		log.Warn(bg, "Scanner stayed busy; the next scan re-reads the file", "file", mf.Path)
	}()
	return nil
}
