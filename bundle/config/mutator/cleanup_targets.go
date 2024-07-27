package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type cleanupTargets struct {
	name string
}

// CleanupTargets cleans up configuration properties before the configuration
// is reported by the 'bundle summary' command.
func CleanupTargets() bundle.Mutator {
	return &cleanupTargets{}
}

func (m *cleanupTargets) Name() string {
	return fmt.Sprintf("Cleanup(%s)", m.name)
}

func (m *cleanupTargets) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	b.Config.Targets = nil
	b.Config.Environments = nil
	return nil
}
