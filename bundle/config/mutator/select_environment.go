package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
)

type selectEnvironment struct {
	name string
}

// SelectEnvironment merges the specified environment into the root configuration.
func SelectEnvironment(name string) bundle.Mutator {
	return &selectEnvironment{
		name: name,
	}
}

func (m *selectEnvironment) Name() string {
	return fmt.Sprintf("SelectEnvironment(%s)", m.name)
}

func (m *selectEnvironment) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if b.Config.Environments == nil {
		return nil, fmt.Errorf("no environments defined")
	}

	// Get specified environment
	env, ok := b.Config.Environments[m.name]
	if !ok {
		return nil, fmt.Errorf("%s: no such environment", m.name)
	}

	// Merge specified environment into root configuration structure.
	err := b.Config.MergeEnvironment(env)
	if err != nil {
		return nil, err
	}

	// Store specified environment in configuration for reference.
	b.Config.Bundle.Environment = m.name

	// Clear environments after loading.
	b.Config.Environments = nil
	return nil, nil
}
