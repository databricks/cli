package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestRestoreOriginalRefs_ScalarMatchesOriginalVariable(t *testing.T) {
	preResolved := dyn.V("${var.catalog}")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{
				"value": dyn.V("main"),
			}),
		}),
	})

	result := restoreOriginalRefs("main", preResolved, resolved)
	assert.Equal(t, "${var.catalog}", result)
}

func TestRestoreOriginalRefs_ScalarDoesNotMatchOriginalVariable(t *testing.T) {
	preResolved := dyn.V("${var.catalog}")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{
				"value": dyn.V("main"),
			}),
		}),
	})

	result := restoreOriginalRefs("staging", preResolved, resolved)
	assert.Equal(t, "staging", result)
}

func TestRestoreOriginalRefs_HardcodedFieldNotRewritten(t *testing.T) {
	preResolved := dyn.V("us-east-1")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"region": dyn.V(map[string]dyn.Value{
				"value": dyn.V("main"),
			}),
		}),
	})

	result := restoreOriginalRefs("main", preResolved, resolved)
	assert.Equal(t, "main", result)
}

func TestRestoreOriginalRefs_InvalidPreResolvedSkipped(t *testing.T) {
	result := restoreOriginalRefs("main", dyn.InvalidValue, dyn.V(map[string]dyn.Value{}))
	assert.Equal(t, "main", result)
}

func TestRestoreOriginalRefs_MapRecursesIntoChildren(t *testing.T) {
	preResolved := dyn.V(map[string]dyn.Value{
		"catalog": dyn.V("${var.catalog}"),
		"region":  dyn.V("hardcoded"),
	})
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{
				"value": dyn.V("main"),
			}),
		}),
	})

	value := map[string]any{
		"catalog": "main",
		"region":  "main",
	}
	result := restoreOriginalRefs(value, preResolved, resolved)
	m := result.(map[string]any)
	assert.Equal(t, "${var.catalog}", m["catalog"])
	assert.Equal(t, "main", m["region"])
}

func TestRestoreFromSiblings_UniqueMatchAtSamePath(t *testing.T) {
	// Sibling has ${var.catalog} at .default resolving to "main".
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{
			"name":    dyn.V("existing"),
			"default": dyn.V("${var.catalog}"),
		}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{"value": dyn.V("main")}),
		}),
	})

	value := map[string]any{"name": "new_param", "default": "main"}
	result := restoreFromSiblings(value, siblings, resolved).(map[string]any)
	assert.Equal(t, "new_param", result["name"])
	assert.Equal(t, "${var.catalog}", result["default"])
}

func TestRestoreFromSiblings_ValueMatchesVariableButAtDifferentPath(t *testing.T) {
	// Sibling uses ${var.some_var} = "5" at .max_retries, NOT at .min_retry_interval.
	// New element has .min_retry_interval = 5. Under the sibling rule, this is NOT
	// substituted because no sibling has a variable at .min_retry_interval.
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

func TestRestoreFromSiblings_AmbiguousAcrossSiblings(t *testing.T) {
	// Two siblings have different variables at the same relative path, both resolve
	// to the same value as the new leaf. Ambiguous → skip.
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

func TestRestoreFromSiblings_SameVariableInMultipleSiblings(t *testing.T) {
	// Multiple siblings use the SAME variable — not ambiguous, one reference.
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{"catalog": dyn.V("${var.catalog}")}),
		dyn.V(map[string]dyn.Value{"catalog": dyn.V("${var.catalog}")}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{"value": dyn.V("main")}),
		}),
	})

	value := map[string]any{"catalog": "main"}
	result := restoreFromSiblings(value, siblings, resolved).(map[string]any)
	assert.Equal(t, "${var.catalog}", result["catalog"])
}

func TestRestoreFromSiblings_NoMatchKeepsHardcoded(t *testing.T) {
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{"default": dyn.V("${var.catalog}")}),
	}
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"catalog": dyn.V(map[string]dyn.Value{"value": dyn.V("main")}),
		}),
	})

	value := map[string]any{"default": "us-west-2"}
	result := restoreFromSiblings(value, siblings, resolved).(map[string]any)
	assert.Equal(t, "us-west-2", result["default"])
}

func TestExtractSequenceParent(t *testing.T) {
	tests := []struct {
		input  string
		parent string
		ok     bool
	}{
		{"resources.jobs.my_job.tasks[*]", "resources.jobs.my_job.tasks", true},
		{"resources.jobs.my_job.parameters[5]", "resources.jobs.my_job.parameters", true},
		{"resources.jobs.my_job.parameters[0]", "resources.jobs.my_job.parameters", true},
		{"resources.jobs.my_job.tags.team", "", false},
		{"resources.jobs.my_job.name", "", false},
		{"resources.jobs.my_job.trigger", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parent, ok := extractSequenceParent(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.parent, parent)
			}
		})
	}
}

func TestStripBracketStars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"resources.jobs.my_job.parameters[*]", "resources.jobs.my_job.parameters"},
		{"resources.jobs.my_job.tasks[*].notebook_task", "resources.jobs.my_job.tasks.notebook_task"},
		{"resources.jobs.my_job.name", "resources.jobs.my_job.name"},
		{"[*].field[*]", ".field"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, stripBracketStars(tt.input))
		})
	}
}

func TestRestoreCompoundInterpolation_SuffixChanged(t *testing.T) {
	preResolved := dyn.V("/mnt/${var.storage_account}/raw/landing")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"storage_account": dyn.V(map[string]dyn.Value{
				"value": dyn.V("devstorageacct"),
			}),
		}),
	})

	result := restoreOriginalRefs("/mnt/devstorageacct/raw/landing_v2", preResolved, resolved)
	assert.Equal(t, "/mnt/${var.storage_account}/raw/landing_v2", result)
}

func TestRestoreCompoundInterpolation_PrefixVariable(t *testing.T) {
	preResolved := dyn.V("${bundle.name}_landing_to_raw")
	resolved := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("analytics_pipeline"),
		}),
	})

	result := restoreOriginalRefs("analytics_pipeline_landing_to_raw_v2", preResolved, resolved)
	assert.Equal(t, "${bundle.name}_landing_to_raw_v2", result)
}

func TestRestoreCompoundInterpolation_MultipleVars(t *testing.T) {
	preResolved := dyn.V("jdbc:sqlserver://${var.db_host}:${var.db_port};database=${var.db_name}")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"db_host": dyn.V(map[string]dyn.Value{"value": dyn.V("dev-sql.example.com")}),
			"db_port": dyn.V(map[string]dyn.Value{"value": dyn.V("1433")}),
			"db_name": dyn.V(map[string]dyn.Value{"value": dyn.V("analytics_dev")}),
		}),
	})

	result := restoreOriginalRefs(
		"jdbc:sqlserver://dev-sql.example.com:5432;database=analytics_dev",
		preResolved, resolved,
	)
	assert.Equal(t, "jdbc:sqlserver://${var.db_host}:5432;database=${var.db_name}", result)
}

func TestRestoreCompoundInterpolation_AllVarsMatch(t *testing.T) {
	preResolved := dyn.V("${var.org}-${bundle.name}-job")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"org": dyn.V(map[string]dyn.Value{"value": dyn.V("acme")}),
		}),
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("etl"),
		}),
	})

	result := restoreOriginalRefs("acme-etl-job", preResolved, resolved)
	assert.Equal(t, "${var.org}-${bundle.name}-job", result)
}

func TestRestoreCompoundInterpolation_NoVarsInOriginal(t *testing.T) {
	preResolved := dyn.V("just_a_plain_string")
	resolved := dyn.V(map[string]dyn.Value{})

	result := restoreOriginalRefs("something_else", preResolved, resolved)
	assert.Equal(t, "something_else", result)
}

func TestRestoreCompoundInterpolation_ValueCompletelyDifferent(t *testing.T) {
	preResolved := dyn.V("${var.org_prefix}-phi-encryption-key")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"org_prefix": dyn.V(map[string]dyn.Value{"value": dyn.V("hc-dev")}),
		}),
	})

	result := restoreOriginalRefs("master-encryption-key-v2", preResolved, resolved)
	assert.Equal(t, "master-encryption-key-v2", result)
}
