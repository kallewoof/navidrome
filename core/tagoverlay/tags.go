// Package tagoverlay turns a user's edit of a song's title, artist credit, composer and album into the raw tags of a
// model.TagOverlay, and stores it so the scanner re-reads the song with it.
package tagoverlay

import (
	"errors"
	"regexp"
	"strings"
)

// Fields is one edit. A nil field is not part of it; an empty one clears the tag (except Title, which a song needs).
type Fields struct {
	Title    *string
	Artist   *string
	Composer *string
	Album    *string
}

var ErrEmptyTitle = errors.New("a song needs a title")

// featSplit matches the separators navidrome's ARTIST mapping splits on (resources/mappings.yaml) that join the
// main artists to the featured ones.
var featSplit = regexp.MustCompile(`(?i)\s+(?:feat\.?|ft\.?)\s+`)

var bankWords = map[string]bool{"append": true, "v2": true, "v3": true, "v4": true, "v4x": true, "v5": true,
	"v6": true, "nt": true, "english": true}

var trailingParen = regexp.MustCompile(`^(.*?)\s*\(([^()]*)\)\s*$`)

// voice splits a featured voice into the singer and its voice bank: "初音ミク Append (Dark)" is 初音ミク with the
// bank "Append (Dark)", "V3 GUMI (Sweet)" is GUMI with "V3 (Sweet)".
func voice(token string) (name, bank string) {
	t := strings.TrimSpace(token)
	paren := ""
	if m := trailingParen.FindStringSubmatch(t); m != nil {
		t, paren = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	}
	words := strings.Fields(t)
	var lead, trail []string
	for len(words) > 1 && bankWords[strings.ToLower(words[0])] {
		lead = append(lead, words[0])
		words = words[1:]
	}
	for len(words) > 1 && bankWords[strings.ToLower(words[len(words)-1])] {
		trail = append([]string{words[len(words)-1]}, trail...)
		words = words[:len(words)-1]
	}
	bankWordsUsed := append(lead, trail...)
	switch {
	case len(bankWordsUsed) > 0 && paren != "":
		bank = strings.Join(bankWordsUsed, " ") + " (" + paren + ")"
	case len(bankWordsUsed) > 0:
		bank = strings.Join(bankWordsUsed, " ")
	default:
		bank = paren
	}
	return strings.Join(words, " "), bank
}

// creditTags are the tags an artist credit owns. "Producer feat. Voice Bank" keeps the banks in ARTIST, names the
// producers and plain voices in ARTISTS (so a voice has one artist page whatever the bank), and credits each voice
// as a performer with its bank, or "vocals", as the sub-role. A credit without "feat." clears ARTISTS and the
// performers, which would otherwise outrank it. VocaDB writes "(Unknown)" for a voice bank it does not know; that
// is no bank, in the credit or the performer role.
func creditTags(credit string) map[string][]string {
	credit = strings.ReplaceAll(strings.TrimSpace(credit), " (Unknown)", "")
	tags := map[string][]string{"artist": {credit}, "artists": {}, "performer": {}}
	if credit == "" {
		tags["artist"] = []string{}
		return tags
	}
	parts := featSplit.Split(credit, 2)
	if len(parts) < 2 {
		return tags
	}
	delete(tags, "performer")
	var names []string
	seen := map[string]bool{}
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, p := range strings.Split(parts[0], ", ") {
		add(strings.TrimSpace(p))
	}
	for _, v := range strings.Split(parts[1], ", ") {
		name, bank := voice(v)
		if name == "" {
			continue
		}
		add(name)
		role := "vocals"
		if bank != "" {
			role = strings.ToLower(bank)
		}
		tags["performer:"+role] = append(tags["performer:"+role], name)
	}
	tags["artists"] = names
	return tags
}

// Merge applies an edit to an overlay's tags: each field in the edit replaces the tags it owns, the others stay.
func Merge(existing map[string][]string, f Fields) (map[string][]string, error) {
	out := map[string][]string{}
	for k, v := range existing {
		out[k] = v
	}
	set := func(owned func(string) bool, tags map[string][]string) {
		for k := range out {
			if owned(k) {
				delete(out, k)
			}
		}
		for k, v := range tags {
			out[k] = v
		}
	}
	single := func(key string, v *string) {
		val := strings.TrimSpace(*v)
		values := []string{}
		if val != "" {
			values = []string{val}
		}
		set(func(k string) bool { return k == key }, map[string][]string{key: values})
	}
	if f.Title != nil {
		if strings.TrimSpace(*f.Title) == "" {
			return nil, ErrEmptyTitle
		}
		single("title", f.Title)
	}
	if f.Artist != nil {
		set(func(k string) bool {
			return k == "artist" || k == "artists" || k == "performer" || strings.HasPrefix(k, "performer:")
		}, creditTags(*f.Artist))
	}
	if f.Composer != nil {
		single("composer", f.Composer)
	}
	if f.Album != nil {
		single("album", f.Album)
	}
	return out, nil
}

// Edited reads an overlay's tags back as the fields a user edited, for showing the current edit.
func Edited(tags map[string][]string) map[string]string {
	out := map[string]string{}
	for field, key := range map[string]string{"title": "title", "artist": "artist", "composer": "composer", "album": "album"} {
		if v, ok := tags[key]; ok {
			out[field] = strings.Join(v, "")
		}
	}
	return out
}
