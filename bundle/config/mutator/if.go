package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
)

type ifMutator struct {
	condition      func(*bundle.Bundle) bool
	onTrueMutator  bundle.Mutator
	onFalseMutator bundle.Mutator
}

func If(
	condition func(*bundle.Bundle) bool,
	onTrueMutator bundle.Mutator,
	onFalseMutator bundle.Mutator,
) bundle.Mutator {
	return &ifMutator{
		condition, onTrueMutator, onFalseMutator,
	}
}

func (m *ifMutator) Apply(ctx context.Context, b *bundle.Bundle) error {
	if m.condition(b) {
		return bundle.Apply(ctx, b, m.onTrueMutator)
	} else {
		return bundle.Apply(ctx, b, m.onFalseMutator)
	}
}

func (m *ifMutator) Name() string {
	return "If"
}
