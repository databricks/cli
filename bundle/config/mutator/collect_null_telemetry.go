package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

// collectNullTelemetry records whether the config has any null value anywhere
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
	b.Metrics.SetBoolValue("null-in-targets", b.Config.NullInTargets())
	return nil
}
