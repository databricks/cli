// Package phases defines build phases as logical groups of [bundle.Mutator] instances.
package phases

import (
	"context"

	"github.com/databricks/bricks/bundle"
)

// This phase type groups mutators that belong to a lifecycle phase.
// It expands into the specific mutators when applied.
type phase struct {
	name     string
	mutators []bundle.Mutator
}

func newPhase(name string, mutators []bundle.Mutator) bundle.Mutator {
	return &phase{
		name:     name,
		mutators: mutators,
	}
}

func (p *phase) Name() string {
	return p.name
}

func (p *phase) Apply(context.Context, *bundle.Bundle) ([]bundle.Mutator, error) {
	return p.mutators, nil
}
