package bundle

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
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

func (m *LogStringMutator) Apply(ctx context.Context, b *Bundle) error {
	cmdio.LogString(ctx, m.message)

	return nil
}
