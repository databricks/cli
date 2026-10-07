package validate

import (
	"fmt"

	"github.com/databricks/cli/libs/structs/structpath"
)

type patternEntry struct {
	key     string
	pattern *structpath.PatternNode
}

// patternSet finds the pattern that matches a concrete path exactly, where wildcards match a single component.
type patternSet map[int][]patternEntry

func newPatternSet(keys []string) (patternSet, error) {
	s := patternSet{}
	for _, k := range keys {
		pattern, err := structpath.ParsePattern(k)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", k, err)
		}
		s[pattern.Len()] = append(s[pattern.Len()], patternEntry{k, pattern})
	}
	return s, nil
}

// find returns the key of the pattern matching p.
func (s patternSet) find(p *structpath.PathNode) (string, bool) {
	for _, e := range s[p.Len()] {
		if p.HasPatternPrefix(e.pattern) {
			return e.key, true
		}
	}
	return "", false
}
