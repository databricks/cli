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

	// Remote changed to a different value — no restoration.
	result := restoreOriginalRefs("staging", preResolved, resolved)
	assert.Equal(t, "staging", result)
}

func TestRestoreOriginalRefs_HardcodedFieldNotRewritten(t *testing.T) {
	// Pre-resolved field was a plain string, not a variable reference.
	preResolved := dyn.V("us-east-1")
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"region": dyn.V(map[string]dyn.Value{
				"value": dyn.V("main"),
			}),
		}),
	})

	// Even though "main" might match a variable, restoreOriginalRefs must NOT
	// rewrite it — the original was hardcoded.
	result := restoreOriginalRefs("main", preResolved, resolved)
	assert.Equal(t, "main", result)
}

func TestRestoreOriginalRefs_InvalidPreResolvedSkipped(t *testing.T) {
	// When preResolved is invalid (field didn't exist in original YAML),
	// restoreOriginalRefs does not modify the value.
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
		"region":  "main", // same value but original was hardcoded — must NOT rewrite
	}
	result := restoreOriginalRefs(value, preResolved, resolved)
	m := result.(map[string]any)
	assert.Equal(t, "${var.catalog}", m["catalog"])
	assert.Equal(t, "main", m["region"])
}

func TestRestoreFromReverseMap_UniqueMatch(t *testing.T) {
	reverseMap := map[any][]string{
		"main": {"${var.catalog}"},
	}
	result := restoreFromReverseMap("main", reverseMap)
	assert.Equal(t, "${var.catalog}", result)
}

func TestRestoreFromReverseMap_AmbiguousSkipped(t *testing.T) {
	reverseMap := map[any][]string{
		"raw_data": {"${var.landing_schema}", "${var.curated_schema}"},
	}
	result := restoreFromReverseMap("raw_data", reverseMap)
	assert.Equal(t, "raw_data", result)
}

func TestRestoreFromReverseMap_NoMatch(t *testing.T) {
	reverseMap := map[any][]string{
		"main": {"${var.catalog}"},
	}
	result := restoreFromReverseMap("us-west-2", reverseMap)
	assert.Equal(t, "us-west-2", result)
}

func TestRestoreFromReverseMap_NestedMap(t *testing.T) {
	reverseMap := map[any][]string{
		"main": {"${var.catalog}"},
		"dev":  {"${bundle.target}"},
	}
	value := map[string]any{
		"catalog":     "main",
		"environment": "dev",
		"region":      "us-west-2",
	}
	result := restoreFromReverseMap(value, reverseMap)
	m := result.(map[string]any)
	assert.Equal(t, "${var.catalog}", m["catalog"])
	assert.Equal(t, "${bundle.target}", m["environment"])
	assert.Equal(t, "us-west-2", m["region"])
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

func TestFieldVariableContext_FieldHasVariable(t *testing.T) {
	cache := map[string]dyn.Value{
		"test.yml": dyn.V(map[string]dyn.Value{
			"name": dyn.V("${var.job_name}"),
		}),
	}
	v, hasCtx := fieldVariableContext(cache, "test.yml", []string{"name"})
	assert.True(t, hasCtx)
	assert.True(t, v.IsValid())
	assert.Equal(t, "${var.job_name}", v.MustString())
}

func TestFieldVariableContext_NoVariablesAnywhere(t *testing.T) {
	cache := map[string]dyn.Value{
		"test.yml": dyn.V(map[string]dyn.Value{
			"name":    dyn.V("hardcoded"),
			"retries": dyn.V(3),
		}),
	}
	_, hasCtx := fieldVariableContext(cache, "test.yml", []string{"name"})
	assert.False(t, hasCtx)
}

func TestFieldVariableContext_ParentHasVariableButFieldDoesNot(t *testing.T) {
	cache := map[string]dyn.Value{
		"test.yml": dyn.V(map[string]dyn.Value{
			"params": dyn.V(map[string]dyn.Value{
				"catalog": dyn.V("${var.catalog}"),
				"region":  dyn.V("us-east-1"),
			}),
		}),
	}
	// Field "params.region" is hardcoded, but parent "params" has a variable sibling.
	// hasContext is true (for Add), but the value is invalid (field itself has no variable).
	v, hasCtx := fieldVariableContext(cache, "test.yml", []string{"params.region"})
	assert.True(t, hasCtx)
	assert.False(t, v.IsValid())
}
