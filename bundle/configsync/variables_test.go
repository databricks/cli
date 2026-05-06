package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

// TestRestoreOriginalRefs_HardcodedFieldNotRewritten fences the Replace safety
// invariant: a hardcoded leaf must never be rewritten to a variable reference
// just because the remote value coincidentally matches a variable elsewhere.
func TestRestoreOriginalRefs_HardcodedFieldNotRewritten(t *testing.T) {
	preResolved := dyn.V("us-east-1")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"region": dyn.V(map[string]dyn.Value{"value": dyn.V("main")}),
		}),
	})
	// Even though "main" matches ${var.region}, restoreOriginalRefs must NOT
	// rewrite it — the original was hardcoded.
	result := restoreOriginalRefs("main", preResolved, resolved)
	assert.Equal(t, "main", result)
}

// TestRestoreFromSiblings_ValueMatchesVariableButDifferentPath fences the Add
// false-positive guard: a new leaf's value matching a variable at a DIFFERENT
// relative path in a sibling must not trigger restoration.
func TestRestoreFromSiblings_ValueMatchesVariableButDifferentPath(t *testing.T) {
	// Sibling uses ${var.retry_count}=5 at .max_retries. New element has
	// .min_retry_interval=5 — coincidental match at a DIFFERENT relative path.
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{
			"task_key":    dyn.V("main"),
			"max_retries": dyn.V("${var.retry_count}"),
		}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"retry_count": dyn.V(map[string]dyn.Value{"value": dyn.V(int64(5))}),
		}),
	})
	value := map[string]any{
		"task_key":           "other",
		"min_retry_interval": int64(5),
	}
	result := restoreFromSiblings(value, siblings, resolved).(map[string]any)
	assert.Equal(t, "other", result["task_key"])
	assert.Equal(t, int64(5), result["min_retry_interval"])
}

// TestRestoreFromSiblings_AmbiguousAcrossSiblings fences the multi-variable
// same-value rule: when two siblings use different variables at the same
// relative path that both resolve to the same value, restoration is skipped.
func TestRestoreFromSiblings_AmbiguousAcrossSiblings(t *testing.T) {
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{"default": dyn.V("${var.landing_schema}")}),
		dyn.V(map[string]dyn.Value{"default": dyn.V("${var.curated_schema}")}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"landing_schema": dyn.V(map[string]dyn.Value{"value": dyn.V("raw_data")}),
			"curated_schema": dyn.V(map[string]dyn.Value{"value": dyn.V("raw_data")}),
		}),
	})
	value := map[string]any{"default": "raw_data"}
	result := restoreFromSiblings(value, siblings, resolved).(map[string]any)
	assert.Equal(t, "raw_data", result["default"])
}

// TestRestoreCompoundInterpolation covers the template alignment algorithm.
// End-to-end coverage (pure ref match, sibling match, non-sequence skip, etc.)
// lives in acceptance/bundle/config-remote-sync/resolve_variables.
func TestRestoreCompoundInterpolation(t *testing.T) {
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"host": dyn.V(map[string]dyn.Value{"value": dyn.V("dev-sql.example.com")}),
			"port": dyn.V(map[string]dyn.Value{"value": dyn.V("1433")}),
			"db":   dyn.V(map[string]dyn.Value{"value": dyn.V("analytics_dev")}),
			"acct": dyn.V(map[string]dyn.Value{"value": dyn.V("acct")}),
		}),
	})

	tests := []struct {
		name     string
		template string
		remote   string
		want     string
	}{
		{
			name:     "suffix change",
			template: "/mnt/${var.acct}/raw/landing",
			remote:   "/mnt/acct/raw/landing_v2",
			want:     "/mnt/${var.acct}/raw/landing_v2",
		},
		{
			name:     "partial variable change preserves others",
			template: "jdbc:sqlserver://${var.host}:${var.port};database=${var.db}",
			remote:   "jdbc:sqlserver://dev-sql.example.com:5432;database=analytics_dev",
			want:     "jdbc:sqlserver://${var.host}:5432;database=${var.db}",
		},
		{
			name:     "completely unrelated value falls back to hardcoded",
			template: "${var.acct}-phi-encryption-key",
			remote:   "master-encryption-key-v2",
			want:     "master-encryption-key-v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := restoreOriginalRefs(tt.remote, dyn.V(tt.template), resolved)
			assert.Equal(t, tt.want, result)
		})
	}
}
