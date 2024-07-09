package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type ifMutator struct {
	condition      func(context.Context, *bundle.Bundle) (bool, error)
	onTrueMutator  bundle.Mutator
	onFalseMutator bundle.Mutator
}

func If(
	condition func(context.Context, *bundle.Bundle) (bool, error),
	onTrueMutator bundle.Mutator,
	onFalseMutator bundle.Mutator,
) bundle.Mutator {
	return &ifMutator{
		condition, onTrueMutator, onFalseMutator,
	}
}

func (m *ifMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	v, err := m.condition(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	if v {
		return bundle.Apply(ctx, b, m.onTrueMutator)
	} else {
		return bundle.Apply(ctx, b, m.onFalseMutator)
	}
}

func (m *ifMutator) Name() string {
	return "If"
}
