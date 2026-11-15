package model

import "time"

// TagOverlay is a user's correction of one song's tags, kept in the database and applied by the scanner over the
// tags it reads from the file, so the music folder stays read-only. Tags holds raw tag names as the scanner reads
// them (lower-case: "title", "artist", "artists", "performer:vocals"). A tag in the overlay replaces every alias of
// that tag in the file, and an empty list removes it.
type TagOverlay struct {
	MediaFileID string
	Tags        map[string][]string
	UpdatedAt   time.Time
	UpdatedBy   string
}

type TagOverlayRepository interface {
	Get(mediaFileID string) (*TagOverlay, error)
	GetAll() ([]TagOverlay, error)
	// GetMany returns the overlays of those of mediaFileIDs that have one, by media file ID.
	GetMany(mediaFileIDs ...string) (map[string]TagOverlay, error)
	Put(overlay *TagOverlay) error
	Delete(mediaFileID string) error
	// MarkForRescan makes the next quick scan of the song's folder re-read the song's file, which a quick scan
	// otherwise skips while the file and its folder are unchanged: it clears the folder's change hash and dates
	// the song's row before any file.
	MarkForRescan(mediaFileID string) error
}
