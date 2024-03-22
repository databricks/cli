package bundle

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

type LogStringMutator struct {
	message string
}

func (d *LogStringMutator) Name() string {
	return "log_string"
}

func LogString(message string) Mutator {
	return &LogStringMutator{
		message: message,
	}
}

func (m *LogStringMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	cmdio.LogString(ctx, m.message)

	return nil
}
