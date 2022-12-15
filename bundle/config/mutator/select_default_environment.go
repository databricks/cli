package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/bricks/bundle"
	"golang.org/x/exp/maps"
)

type selectDefaultEnvironment struct{}

// SelectDefaultEnvironment merges the default environment into the root configuration.
func SelectDefaultEnvironment() bundle.Mutator {
	return &selectDefaultEnvironment{}
}

func (m *selectDefaultEnvironment) Name() string {
	return "SelectDefaultEnvironment"
}

func (m *selectDefaultEnvironment) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if len(b.Config.Environments) == 0 {
		return nil, fmt.Errorf("no environments defined")
	}

	// One environment means there's only one default.
	names := maps.Keys(b.Config.Environments)
	if len(names) == 1 {
		return []bundle.Mutator{SelectEnvironment(names[0])}, nil
	}

	// Multiple environments means we look for the `default` flag.
	var defaults []string
	for _, name := range names {
		if b.Config.Environments[name].Default {
			defaults = append(defaults, name)
		}
	}

	// It is invalid to have multiple environments with the `default` flag set.
	if len(defaults) > 1 {
		return nil, fmt.Errorf("multiple environments are marked as default (%s)", strings.Join(defaults, ", "))
	}

	// If no environment has the `default` flag set, ask the user to specify one.
	if len(defaults) == 0 {
		return nil, fmt.Errorf("please specify environment")
	}

	// One default remaining.
	return []bundle.Mutator{SelectEnvironment(defaults[0])}, nil
}
