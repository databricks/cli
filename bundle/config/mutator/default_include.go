package mutator

import (
	"github.com/databricks/bricks/bundle/config"
	"golang.org/x/exp/slices"
)

type defineDefaultInclude struct {
	include []string
}

// DefineDefaultInclude sets the list of includes to a default if it hasn't been set.
func DefineDefaultInclude() Mutator {
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

func (m *defineDefaultInclude) Apply(root *config.Root) ([]Mutator, error) {
	if len(root.Include) == 0 {
		root.Include = slices.Clone(m.include)
	}
	return nil, nil
}
