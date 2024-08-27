package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
	isset := false
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		pv, _ := dyn.Get(v, "sync.paths")

		// If the sync paths field is already set, do nothing.
		// We know it is set if its value is either a nil or a sequence (empty or not).
		switch pv.Kind() {
		case dyn.KindNil, dyn.KindSequence:
			isset = true
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// If the sync paths field is already set, do nothing.
	if isset {
		return nil
	}

	// Set the sync paths to the default value.
	b.Config.Sync.Paths = []string{"."}
	return nil
}
