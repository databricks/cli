package interpolation

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

// LookupFunction returns the value to rewrite a path expression to.
type LookupFunction func(path string, depends map[string]string) (string, error)

// ErrSkipInterpolation can be used to fall through from [LookupFunction].
var ErrSkipInterpolation = errors.New("skip interpolation")

// DefaultLookup looks up the specified path in the map.
// It returns an error if it doesn't exist.
func DefaultLookup(path string, lookup map[string]string) (string, error) {
	v, ok := lookup[path]
	if !ok {
		return "", fmt.Errorf("expected to find value for path: %s", path)
	}
	return v, nil
}

func pathPrefixMatches(prefix []string, path string) bool {
	parts := strings.Split(path, Delimiter)
	return len(parts) >= len(prefix) && slices.Compare(prefix, parts[0:len(prefix)]) == 0
}

// ExcludeLookupsInPath is a lookup function that skips lookups for the specified path.
func ExcludeLookupsInPath(exclude ...string) LookupFunction {
	return func(path string, lookup map[string]string) (string, error) {
		if pathPrefixMatches(exclude, path) {
			return "", ErrSkipInterpolation
		}

		return DefaultLookup(path, lookup)
	}
}

// IncludeLookupsInPath is a lookup function that limits lookups to the specified path.
func IncludeLookupsInPath(include ...string) LookupFunction {
	return func(path string, lookup map[string]string) (string, error) {
		if !pathPrefixMatches(include, path) {
			return "", ErrSkipInterpolation
		}

		return DefaultLookup(path, lookup)
	}
}
