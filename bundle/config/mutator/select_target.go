package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
)

type selectTarget struct {
	name string
}

// SelectTarget merges the specified target into the root configuration.
func SelectTarget(name string) bundle.Mutator {
	return &selectTarget{
		name: name,
	}
}

func (m *selectTarget) Name() string {
	return fmt.Sprintf("SelectTarget(%s)", m.name)
}

func (m *selectTarget) Apply(_ context.Context, b *bundle.Bundle) error {
	if b.Config.Targets == nil {
		return fmt.Errorf("no targets defined")
	}

	// Get specified target
	env, ok := b.Config.Targets[m.name]
	if !ok {
		return fmt.Errorf("%s: no such target", m.name)
	}

	// Merge specified target into root configuration structure.
	err := b.Config.MergeTarget(env)
	if err != nil {
		return err
	}

	// Store specified target in configuration for reference.
	b.Config.Bundle.Target = m.name

	// We do this for backward compatibility.
	// TODO: remove when Environments section is not supported anymore.
	b.Config.Bundle.Environment = b.Config.Bundle.Target

	// Clear targets after loading.
	b.Config.Targets = nil
	b.Config.Environments = nil

	return nil
}
