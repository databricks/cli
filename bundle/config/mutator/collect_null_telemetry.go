package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

// collectNullTelemetry records per-category flags for null values found anywhere
// under the "targets" section, as authored. The signal is computed at load time
// (before normalization drops nulls on scalar-typed fields) and accumulated across
// all included files; this mutator just surfaces it as telemetry.
type collectNullTelemetry struct{}

func CollectNullTelemetry() bundle.Mutator {
	return &collectNullTelemetry{}
}

func (*collectNullTelemetry) Name() string {
	return "CollectNullTelemetry"
}

func (*collectNullTelemetry) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	info := b.Config.NullInTargets()
	b.Metrics.SetBoolValue("null-in-targets", info.Any())
	b.Metrics.SetBoolValue("null-in-targets.scalar", info.Scalar)
	b.Metrics.SetBoolValue("null-in-targets.complex", info.Complex)
	b.Metrics.SetBoolValue("null-in-targets.map-key", info.MapKey)
	b.Metrics.SetBoolValue("null-in-targets.array-index", info.ArrayIndex)
	b.Metrics.SetBoolValue("null-in-targets.resource", info.Resource)
	return nil
}
