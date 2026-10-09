package structvar

import (
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/structs/structpath"
)

const maxSuggestionDistance = 2

// LevenshteinDistance computes the edit distance between two strings.
func LevenshteinDistance(a, b string) int {
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

// SuggestKeys returns the keys whose edit distance from name is at most
// maxSuggestionDistance, ordered by increasing distance. It is used to build
// "did you mean" hints for a key that was not found in a map.
func SuggestKeys(keys []string, name string) []string {
	type candidate struct {
		key  string
		dist int
	}

	var candidates []candidate
	for _, key := range keys {
		d := LevenshteinDistance(name, key)
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

// DidYouMean formats a suggestion clause like `, did you mean "x"?` (or, for
// multiple candidates, `, did you mean one of: "x", "y"?`). It returns an empty
// string when there are no suggestions.
func DidYouMean(suggestions []string) string {
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

// ReplaceKey returns reference with the component matching failedKey swapped for
// replacement, or just replacement if reference can't be parsed or has no match.
func ReplaceKey(reference, failedKey, replacement string) string {
	p, err := structpath.ParsePath(reference)
	if err != nil {
		return replacement
	}
	var out *structpath.PathNode
	replaced := false
	for _, n := range p.AsSlice() {
		if k, ok := n.StringKey(); ok {
			if k == failedKey && !replaced {
				k = replacement
				replaced = true
			}
			out = structpath.NewStringKey(out, k)
		} else if i, ok := n.Index(); ok {
			out = structpath.NewIndex(out, i)
		}
	}
	if !replaced {
		return replacement
	}
	return out.String()
}
