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
	if info.Scalar {
		b.Metrics.SetBoolValue("null-in-targets-scalar", true)
	}
	if info.Complex {
		b.Metrics.SetBoolValue("null-in-targets-complex", true)
	}
	if info.MapKey {
		b.Metrics.SetBoolValue("null-in-targets-key", true)
	}
	if info.ArrayIndex {
		b.Metrics.SetBoolValue("null-in-targets-index", true)
	}
	if info.Resource {
		b.Metrics.SetBoolValue("null-in-targets-resource", true)
	}
	return nil
}
