package dynvar

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/textutil"
)

// SuggestPath walks the reference path against root, correcting mistyped key
// components via fuzzy matching. Returns the corrected path string, or "" if
// the path cannot be corrected.
func SuggestPath(root dyn.Value, refPath dyn.Path) string {
	cur := root
	suggested := dyn.EmptyPath

	for _, c := range refPath {
		if c.Key() != "" {
			m, ok := cur.AsMap()
			if !ok {
				return ""
			}

			key := c.Key()
			if v, ok := m.GetByString(key); ok {
				suggested = suggested.Append(dyn.Key(key))
				cur = v
				continue
			}

			// Collect candidate keys for fuzzy matching.
			pairs := m.Pairs()
			candidates := make([]string, len(pairs))
			for i, p := range pairs {
				candidates[i] = p.Key.MustString()
			}

			match, _ := textutil.ClosestMatch(key, candidates)
			if match == "" {
				return ""
			}

			v, _ := m.GetByString(match)
			suggested = suggested.Append(dyn.Key(match))
			cur = v
		} else {
			seq, ok := cur.AsSequence()
			if !ok || c.Index() < 0 || c.Index() >= len(seq) {
				return ""
			}
			suggested = suggested.Append(dyn.Index(c.Index()))
			cur = seq[c.Index()]
		}
	}

	return suggested.String()
}
