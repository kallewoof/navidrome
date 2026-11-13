package model

import "time"

type Annotations struct {
	PlayCount     int64      `structs:"play_count"     json:"playCount,omitempty"`
	PlayDate      *time.Time `structs:"play_date"      json:"playDate,omitempty" `
	Rating        int        `structs:"rating"         json:"rating,omitempty"   `
	RatedAt       *time.Time `structs:"rated_at"       json:"ratedAt,omitempty"  `
	Starred       bool       `structs:"starred"        json:"starred,omitempty"  `
	StarredAt     *time.Time `structs:"starred_at"     json:"starredAt,omitempty"`
	AverageRating float64    `structs:"average_rating" json:"averageRating,omitempty"`
	SkipCount     int64      `structs:"skip_count"     json:"skipCount,omitempty"`
	CompleteCount int64      `structs:"complete_count" json:"completeCount,omitempty"`
}

// PlaybackOutcome is how a listen ended, counted per user and item next to the play count.
type PlaybackOutcome int

const (
	// PlaybackSkipped is a stop before the play-count threshold.
	PlaybackSkipped PlaybackOutcome = iota + 1
	// PlaybackCompleted is a stop at 90% of the song or later.
	PlaybackCompleted
)

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
	ReassignAnnotation(prevID string, newID string) error
}
