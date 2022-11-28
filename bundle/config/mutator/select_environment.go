package mutator

import (
	"fmt"

	"github.com/databricks/bricks/bundle/config"
)

type selectEnvironment struct {
	name string
}

// SelectEnvironment merges the specified environment into the root configuration.
func SelectEnvironment(name string) Mutator {
	return &selectEnvironment{
		name: name,
	}
}

func (m *selectEnvironment) Name() string {
	return fmt.Sprintf("SelectEnvironment(%s)", m.name)
}

func (m *selectEnvironment) Apply(root *config.Root) ([]Mutator, error) {
	if root.Environments == nil {
		return nil, fmt.Errorf("no environments defined")
	}

	// Get specified environment
	env, ok := root.Environments[m.name]
	if !ok {
		return nil, fmt.Errorf("%s: no such environment", m.name)
	}

	// Merge specified environment into root configuration structure.
	err := root.MergeEnvironment(env)
	if err != nil {
		return nil, err
	}

	// Store specified environment in configuration for reference.
	root.Bundle.Environment = m.name

	// Clear environments after loading.
	root.Environments = nil
	return nil, nil
}
