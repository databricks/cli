package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type syncDefaultPath struct{}

// SyncDefaultPath configures the default sync path to be equal to the bundle root.
func SyncDefaultPath() bundle.Mutator {
	return &syncDefaultPath{}
}

func (m *syncDefaultPath) Name() string {
	return "SyncDefaultPath"
}

func (m *syncDefaultPath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// If the sync paths field is already set, do nothing.
	// We know it is set if it is a sequence (empty or not) or a reference.
	if b.Config.Sync.Paths != nil || b.Config.IsReference("sync.paths") {
		return nil
	}

	// Set the sync paths to the default value.
	b.Config.Sync.Paths = []string{"."}
	return nil
}
