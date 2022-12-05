package interpolation

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

// LookupFunction returns the value to rewrite a path expression to.
type LookupFunction func(path string, depends map[string]string) (string, error)

// DefaultLookup looks up the specified path in the map.
// It returns an error if it doesn't exist.
func DefaultLookup(path string, lookup map[string]string) (string, error) {
	v, ok := lookup[path]
	if !ok {
		return "", fmt.Errorf("expected to find value for path: %s", path)
	}
	return v, nil
}

// ExcludeLookupsInPath is a lookup function that skips lookups for the specified path.
func ExcludeLookupsInPath(exclude ...string) LookupFunction {
	return func(path string, lookup map[string]string) (string, error) {
		parts := strings.Split(path, Delimiter)

		// Skip interpolation of this path.
		if len(parts) >= len(exclude) && slices.Compare(exclude, parts[0:len(exclude)]) == 0 {
			return fmt.Sprintf("${%s}", path), nil
		}

		return DefaultLookup(path, lookup)
	}
}
