package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
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
	result := restoreOriginalRefs("main", preResolved, resolved, allowAllVariables)
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
	result := restoreFromSiblings(value, siblings, resolved, allowAllVariables).(map[string]any)
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
	result := restoreFromSiblings(value, siblings, resolved, allowAllVariables).(map[string]any)
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
			result := restoreOriginalRefs(tt.remote, dyn.V(tt.template), resolved, allowAllVariables)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNewCrossTargetGuard_Safe(t *testing.T) {
	strDefault := func(s string) *variable.TargetVariable {
		return &variable.TargetVariable{Default: s}
	}
	cfg := &config.Root{
		Variables: map[string]*variable.Variable{
			"root_default":    {Default: "v"},
			"root_lookup":     {Lookup: &variable.Lookup{Cluster: "c"}},
			"all_targets":     {},
			"dev_only":        {},
			"env_or_file_fed": {},
		},
		Targets: map[string]*config.Target{
			"dev": {Variables: map[string]*variable.TargetVariable{
				"all_targets": strDefault("dev-v"),
				"dev_only":    strDefault("dev-v"),
			}},
			"prod": {Variables: map[string]*variable.TargetVariable{
				"all_targets": strDefault("prod-v"),
			}},
		},
	}

	g := newCrossTargetGuard(cfg)
	assert.True(t, g.multiTarget)
	assert.True(t, g.safe["root_default"])
	assert.True(t, g.safe["root_lookup"])
	assert.True(t, g.safe["all_targets"])
	assert.False(t, g.safe["dev_only"])
	assert.False(t, g.safe["env_or_file_fed"])
}

func TestCrossTargetGuard_AllowFor(t *testing.T) {
	targetsRoot := dyn.V(map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				"foo": dyn.V(map[string]dyn.Value{
					"tasks": dyn.V([]dyn.Value{dyn.V(map[string]dyn.Value{"task_key": dyn.V("main")})}),
				}),
			}),
		}),
		"targets": dyn.V(map[string]dyn.Value{
			"dev": dyn.V(map[string]dyn.Value{
				"resources": dyn.V(map[string]dyn.Value{
					"jobs": dyn.V(map[string]dyn.Value{
						"foo": dyn.V(map[string]dyn.Value{
							"tags":  dyn.V(map[string]dyn.Value{"env": dyn.V("x")}),
							"tasks": dyn.V([]dyn.Value{dyn.V(map[string]dyn.Value{"task_key": dyn.V("extra")})}),
						}),
					}),
				}),
			}),
		}),
	})

	multi := &crossTargetGuard{targetsRoot: targetsRoot, multiTarget: true, safe: map[string]bool{"safe_var": true}}
	single := &crossTargetGuard{targetsRoot: targetsRoot, multiTarget: false, safe: map[string]bool{}}

	tests := []struct {
		name       string
		guard      *crossTargetGuard
		candidates []string
		variable   string
		want       bool
	}{
		{
			name:       "single target allows everything",
			guard:      single,
			candidates: []string{"resources.jobs.foo.tasks[*]"},
			variable:   "dev_only",
			want:       true,
		},
		{
			name:       "shared destination allows safe variable",
			guard:      multi,
			candidates: []string{"resources.jobs.foo.tasks[*]", "targets.dev.resources.jobs.foo.tasks[*]"},
			variable:   "safe_var",
			want:       true,
		},
		{
			name:       "shared destination rejects unsafe variable",
			guard:      multi,
			candidates: []string{"resources.jobs.foo.tasks[*]", "targets.dev.resources.jobs.foo.tasks[*]"},
			variable:   "dev_only",
			want:       false,
		},
		{
			name:       "target override destination allows unsafe variable",
			guard:      multi,
			candidates: []string{"resources.jobs.foo.tags.env", "targets.dev.resources.jobs.foo.tags.env"},
			variable:   "dev_only",
			want:       true,
		},
		{
			// Pins the documented destinationInTarget limitation: the
			// file-local index of an element defined only in the target
			// override collides with the shared sequence, so the change
			// classifies as shared and the strict rule applies.
			name:       "indexed override element falls back to strict rule",
			guard:      multi,
			candidates: []string{"resources.jobs.foo.tasks[0].task_key", "targets.dev.resources.jobs.foo.tasks[0].task_key"},
			variable:   "dev_only",
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allow := tt.guard.allowFor(tt.candidates)
			assert.Equal(t, tt.want, allow(tt.variable))
		})
	}
}

func TestVarRefsAllowed(t *testing.T) {
	allow := func(name string) bool { return name == "safe_var" }

	tests := []struct {
		s    string
		want bool
	}{
		{"${var.safe_var}", true},
		{"${var.dev_only}", false},
		{"${bundle.target}", true},
		{"${resources.pipelines.p.id}", true},
		{"/mnt/${var.safe_var}/${bundle.target}/raw", true},
		{"/mnt/${var.safe_var}/${var.dev_only}/raw", false},
		{"no refs at all", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.want, varRefsAllowed(tt.s, allow))
		})
	}
}

// TestRestoreFromSiblings_GuardBlocksCrossTargetVariable fences the Add-path
// guard: a unique sibling match naming a variable other targets can't resolve
// must stay hardcoded instead of leaking into a shared file.
func TestRestoreFromSiblings_GuardBlocksCrossTargetVariable(t *testing.T) {
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{"default": dyn.V("${var.dev_only}")}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"dev_only": dyn.V(map[string]dyn.Value{"value": dyn.V("dev_data")}),
		}),
	})
	allow := func(name string) bool { return false }

	value := map[string]any{"default": "dev_data"}
	result := restoreFromSiblings(value, siblings, resolved, allow).(map[string]any)
	assert.Equal(t, "dev_data", result["default"])
}

// TestRestoreOriginalRefs_GuardBlocksFallback fences the Replace-fallback
// guard: re-targeting a pure ${var.X} field to a variable other targets can't
// resolve must fall back to the hardcoded value.
func TestRestoreOriginalRefs_GuardBlocksFallback(t *testing.T) {
	preResolved := dyn.V("${var.region}")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"region":   dyn.V(map[string]dyn.Value{"value": dyn.V("us-west-1")}),
			"dev_only": dyn.V(map[string]dyn.Value{"value": dyn.V("dev_data")}),
		}),
	})
	allow := func(name string) bool { return name == "region" }

	result := restoreOriginalRefs("dev_data", preResolved, resolved, allow)
	assert.Equal(t, "dev_data", result)
}
