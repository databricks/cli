package bundle

import (
	"context"

	"github.com/databricks/cli/libs/diag"
)

type ifMutator struct {
	condition      func(context.Context, *Bundle) (bool, error)
	onTrueMutator  Mutator
	onFalseMutator Mutator
}

func If(
	condition func(context.Context, *Bundle) (bool, error),
	onTrueMutator Mutator,
	onFalseMutator Mutator,
) Mutator {
	return &ifMutator{
		condition, onTrueMutator, onFalseMutator,
	}
}

func (m *ifMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	v, err := m.condition(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	if v {
		return Apply(ctx, b, m.onTrueMutator)
	} else {
		return Apply(ctx, b, m.onFalseMutator)
	}
}

func (m *ifMutator) Name() string {
	return "If"
}
