package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// collectNullTelemetry records whether the config has any null value anywhere
// under the "targets" section. It must run before the target overrides are
// merged, since merging drops the "targets" section from the config tree.
//
// It inspects the normalized config, which is what the merge itself operates on.
// Normalization drops nulls on scalar-typed fields (e.g. workspace.host: with no
// value), so those are not counted; nulls on map, sequence, and struct-typed
// fields (e.g. resources:, variables.foo:) survive and are the ones the merge sees.
type collectNullTelemetry struct{}

func CollectNullTelemetry() bundle.Mutator {
	return &collectNullTelemetry{}
}

func (*collectNullTelemetry) Name() string {
	return "CollectNullTelemetry"
}

func (*collectNullTelemetry) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	hasNull := false

	targets := b.Config.Value().Get("targets")
	if targets.Kind() == dyn.KindMap {
		err := dyn.WalkReadOnly(targets, func(p dyn.Path, v dyn.Value) error {
			if v.Kind() == dyn.KindNil {
				hasNull = true
				return dyn.ErrSkip
			}
			return nil
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	b.Metrics.SetBoolValue("null-in-targets", hasNull)
	return nil
}
