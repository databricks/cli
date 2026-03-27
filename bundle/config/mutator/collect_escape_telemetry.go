package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Temporary: this mutator collects telemetry on escape patterns ($${} and \${})
// in bundle configs. It will be removed once we align on the escape syntax.
type collectEscapeTelemetry struct{}

func CollectEscapeTelemetry() bundle.Mutator {
	return &collectEscapeTelemetry{}
}

func (*collectEscapeTelemetry) Name() string {
	return "CollectEscapeTelemetry"
}

func (*collectEscapeTelemetry) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	hasDoubleDollar := false
	hasBackslashDollar := false

	_, err := dyn.Walk(b.Config.Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		s, ok := v.AsString()
		if !ok {
			return v, nil
		}

		if !hasDoubleDollar && strings.Contains(s, "$${") {
			hasDoubleDollar = true
		}
		if !hasBackslashDollar && strings.Contains(s, "\\${") {
			hasBackslashDollar = true
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if hasDoubleDollar {
		b.Metrics.SetBoolValue("config_has_double_dollar", true)
	}
	if hasBackslashDollar {
		b.Metrics.SetBoolValue("config_has_backslash_dollar", true)
	}

	return nil
}
