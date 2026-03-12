package dynvar

import (
	"github.com/databricks/cli/libs/dyn"
)

// levenshtein computes the edit distance between two strings using single-row DP.
func levenshtein(a, b string) int {
	if len(a) < len(b) {
		a, b = b, a
	}

	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}

	for i := range len(a) {
		prev := row[0]
		row[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			tmp := row[j+1]
			row[j+1] = min(
				row[j+1]+1,  // deletion
				row[j]+1,    // insertion
				prev+cost,   // substitution
			)
			prev = tmp
		}
	}

	return row[len(b)]
}

// closestKeyMatch finds the candidate with the smallest edit distance to key,
// within a threshold of min(3, max(1, len(key)/2)). Returns ("", 0) if no
// candidate is within threshold.
func closestKeyMatch(key string, candidates []string) (string, int) {
	threshold := min(3, max(1, len(key)/2))
	bestDist := threshold + 1
	bestMatch := ""

	for _, c := range candidates {
		d := levenshtein(key, c)
		if d < bestDist {
			bestDist = d
			bestMatch = c
		}
	}

	if bestMatch == "" {
		return "", 0
	}
	return bestMatch, bestDist
}

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

			match, _ := closestKeyMatch(key, candidates)
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
