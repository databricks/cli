package mutator

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"golang.org/x/exp/slices"
)

type processRootIncludes struct{}

func ProcessRootIncludes() Mutator {
	return &processRootIncludes{}
}

func (m *processRootIncludes) Name() string {
	return "ProcessRootIncludes"
}

func (m *processRootIncludes) Apply(root *config.Root) ([]Mutator, error) {
	var out []Mutator

	// Map with files we've already seen to avoid loading them twice.
	var seen = map[string]bool{
		bundle.ConfigFile: true,
	}

	// For each glob, find all files to load.
	// Ordering of the list of globs is maintained in the output.
	// For matches that appear in multiple globs, only the first is kept.
	for _, entry := range root.Include {
		// Include paths must be relative.
		if filepath.IsAbs(entry) {
			return nil, fmt.Errorf("%s: includes must be relative paths", entry)
		}

		// Anchor includes to the bundle root path.
		matches, err := filepath.Glob(filepath.Join(root.Path, entry))
		if err != nil {
			return nil, err
		}

		// Filter matches to ones we haven't seen yet.
		var includes []string
		for _, match := range matches {
			rel, err := filepath.Rel(root.Path, match)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = true
			includes = append(includes, rel)
		}

		// Add matches to list of mutators to return.
		slices.Sort(includes)
		for _, include := range includes {
			out = append(out, ProcessInclude(filepath.Join(root.Path, include), include))
		}
	}

	return out, nil
}
