package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Temporary: this mutator collects telemetry on escape patterns ($${}, $$, \${}, \$)
// in bundle configs. It will be removed once we align on the escape syntax.
type collectEscapeTelemetry struct{}

func CollectEscapeTelemetry() bundle.Mutator {
	return &collectEscapeTelemetry{}
}

func (*collectEscapeTelemetry) Name() string {
	return "CollectEscapeTelemetry"
}

func (*collectEscapeTelemetry) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var hasDoubleDollarBrace, hasDoubleDollar, hasBackslashDollarBrace, hasBackslashDollar bool

	_, err := dyn.Walk(b.Config.Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		s, ok := v.AsString()
		if !ok {
			return v, nil
		}

		if !hasDoubleDollarBrace && strings.Contains(s, "$${") {
			hasDoubleDollarBrace = true
		}
		if !hasDoubleDollar && containsDoubleDollarWithoutBrace(s) {
			hasDoubleDollar = true
		}
		if !hasBackslashDollarBrace && strings.Contains(s, "\\${") {
			hasBackslashDollarBrace = true
		}
		if !hasBackslashDollar && containsBackslashDollarWithoutBrace(s) {
			hasBackslashDollar = true
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if hasDoubleDollarBrace {
		b.Metrics.SetBoolValue("config_has_double_dollar_brace", true)
	}
	if hasDoubleDollar {
		b.Metrics.SetBoolValue("config_has_double_dollar", true)
	}
	if hasBackslashDollarBrace {
		b.Metrics.SetBoolValue("config_has_backslash_dollar_brace", true)
	}
	if hasBackslashDollar {
		b.Metrics.SetBoolValue("config_has_backslash_dollar", true)
	}

	return nil
}

// containsDoubleDollarWithoutBrace returns true if s contains "$$" not followed by "{".
func containsDoubleDollarWithoutBrace(s string) bool {
	for i := range len(s) - 1 {
		if s[i] == '$' && s[i+1] == '$' {
			if i+2 >= len(s) || s[i+2] != '{' {
				return true
			}
		}
	}
	return false
}

// containsBackslashDollarWithoutBrace returns true if s contains "\$" not followed by "{".
func containsBackslashDollarWithoutBrace(s string) bool {
	for i := range len(s) - 1 {
		if s[i] == '\\' && s[i+1] == '$' {
			if i+2 >= len(s) || s[i+2] != '{' {
				return true
			}
		}
	}
	return false
}
