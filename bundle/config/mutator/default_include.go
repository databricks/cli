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
	return &defineDefaultInclude{
		// When we support globstar we can collapse below into a single line.
		include: []string{
			// Load YAML files in the same directory.
			"*.yml",
			// Load YAML files in subdirectories.
			"*/*.yml",
		},
	}
}

func (m *defineDefaultInclude) Name() string {
	return "DefineDefaultInclude"
}

func (m *defineDefaultInclude) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if len(b.Config.Include) == 0 {
		b.Config.Include = slices.Clone(m.include)
	}
	return nil, nil
}
