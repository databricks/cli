package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"golang.org/x/exp/slices"
)

type defineDefaultInclude struct {
	include []string
}

// DefineDefaultInclude sets the list of includes to a default if it hasn't been set.
func DefineDefaultInclude() bundle.Mutator {
	var includePaths = []string{
		// When we support globstar we can collapse below into a single line.
		// Load YAML files in the same directory.
		"*.yml",
		// Load YAML files in subdirectories.
		"*/*.yml",
	}
	return &defineDefaultInclude{
		include: append(includePaths, bundle.GetExtraIncludePaths()...),
	}
}

func (m *defineDefaultInclude) Name() string {
	return "DefineDefaultInclude"
}

func (m *defineDefaultInclude) Apply(_ context.Context, b *bundle.Bundle) error {
	if len(b.Config.Include) == 0 {
		b.Config.Include = slices.Clone(m.include)
	}
	return nil
}
