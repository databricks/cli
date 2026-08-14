package dyn

import (
	"fmt"
	"slices"
	"strings"
)

const maxSuggestionDistance = 2

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use a single row for the DP table.
	prev := make([]int, len(b)+1)
	for j := range len(b) + 1 {
		prev[j] = j
	}

	for i := range len(a) {
		curr := make([]int, len(b)+1)
		curr[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			curr[j+1] = min(
				curr[j]+1,    // insertion
				prev[j+1]+1,  // deletion
				prev[j]+cost, // substitution
			)
		}
		prev = curr
	}

	return prev[len(b)]
}

// suggestKeys returns the keys in m whose edit distance from name is at most
// maxSuggestionDistance, ordered by increasing distance. It is used to build
// "did you mean" hints for a key that was not found in the map.
func suggestKeys(m Mapping, name string) []string {
	type candidate struct {
		key  string
		dist int
	}

	var candidates []candidate
	for _, kv := range m.Keys() {
		key := kv.MustString()
		d := levenshteinDistance(name, key)
		if d <= maxSuggestionDistance {
			candidates = append(candidates, candidate{key, d})
		}
	}

	slices.SortStableFunc(candidates, func(a, b candidate) int {
		return a.dist - b.dist
	})

	suggestions := make([]string, len(candidates))
	for i, c := range candidates {
		suggestions[i] = c.key
	}
	return suggestions
}

// didYouMean formats a suggestion clause like `, did you mean "x"?` (or, for
// multiple candidates, `, did you mean one of: "x", "y"?`). It returns an empty
// string when there are no suggestions.
func didYouMean(suggestions []string) string {
	switch len(suggestions) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(", did you mean %q?", suggestions[0])
	default:
		quoted := make([]string, len(suggestions))
		for i, s := range suggestions {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		return fmt.Sprintf(", did you mean one of: %s?", strings.Join(quoted, ", "))
	}
}

// didYouMeanReferences formats a "did you mean" block listing each suggestion as
// a full drop-in reference on its own line, with only the failed segment swapped.
// Returns "" when there are no suggestions.
func didYouMeanReferences(reference, failedKey string, suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	lines := make([]string, len(suggestions))
	for i, s := range suggestions {
		lines[i] = "  ${" + replaceKey(reference, failedKey, s) + "}"
	}
	return "\n\ndid you mean:\n" + strings.Join(lines, "\n")
}

// replaceKey returns reference with the component matching failedKey swapped for
// replacement, or just replacement if reference can't be parsed or has no match.
func replaceKey(reference, failedKey, replacement string) string {
	p, err := NewPathFromString(reference)
	if err != nil {
		return replacement
	}
	for i, c := range p {
		if c.Key() == failedKey {
			out := p.Append()
			out[i] = Key(replacement)
			return out.String()
		}
	}
	return replacement
}
