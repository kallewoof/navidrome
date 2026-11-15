package metadata

import (
	"strings"

	"github.com/navidrome/navidrome/model"
)

// ApplyOverlay puts a model.TagOverlay's tags over the raw tags read from a file, returning a new map. Each tag the
// overlay sets replaces every alias of that tag in raw ("artist" also drops "tpe1" and "©art"; "performer:vocals"
// drops every performer credit), because the mapping merges aliases and a wrong value left under one would survive
// next to the correction. An empty value list removes the tag.
func ApplyOverlay(raw model.RawTags, overlay map[string][]string) model.RawTags {
	if len(overlay) == 0 {
		return raw
	}
	out := make(model.RawTags, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	mappings := model.TagMappings()
	for key := range overlay {
		name, pair := tagNameOf(strings.ToLower(key), mappings)
		aliases := append([]string{string(name)}, mappings[name].Aliases...)
		for k := range out {
			lk := strings.ToLower(k)
			for _, a := range aliases {
				if lk == a || (pair && strings.HasPrefix(lk, a+":")) {
					delete(out, k)
					break
				}
			}
		}
	}
	for key, values := range overlay {
		if len(values) > 0 {
			out[strings.ToLower(key)] = values
		}
	}
	return out
}

// tagNameOf is the mapped tag a raw key sets: itself, or for a pair tag's "name:role" form, the name.
func tagNameOf(key string, mappings map[model.TagName]model.TagConf) (model.TagName, bool) {
	if c, ok := mappings[model.TagName(key)]; ok {
		return model.TagName(key), c.Type == model.TagTypePair
	}
	if base, _, found := strings.Cut(key, ":"); found {
		if c, ok := mappings[model.TagName(base)]; ok && c.Type == model.TagTypePair {
			return model.TagName(base), true
		}
	}
	return model.TagName(key), false
}
