package bundle

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
)

type PrintMutator struct {
	message string
}

func (d *PrintMutator) Name() string {
	return "deferred"
}

func Print(message string) Mutator {
	return &PrintMutator{
		message: message,
	}
}

func (m *PrintMutator) Apply(ctx context.Context, b *Bundle) error {
	cmdio.LogString(ctx, m.message)

	return nil
}
