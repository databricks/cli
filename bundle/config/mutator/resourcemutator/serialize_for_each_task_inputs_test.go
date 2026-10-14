package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forEachTestBundle builds a bundle with a single job whose for_each_task.inputs
// is the given value, plus the given variables.
func forEachTestBundle(inputs string, vars map[string]*variable.Variable) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Variables: vars,
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "process_tables",
									ForEachTask: &jobs.ForEachTask{
										Inputs: inputs,
										Task: jobs.Task{
											TaskKey: "load_table",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func forEachInputs(t *testing.T, b *bundle.Bundle) string {
	t.Helper()
	tasks := b.Config.Resources.Jobs["my_job"].Tasks
	require.Len(t, tasks, 1)
	require.NotNil(t, tasks[0].ForEachTask)
	return tasks[0].ForEachTask.Inputs
}

func TestSerializeForEachTaskInputs(t *testing.T) {
	complexValue := []any{
		map[string]any{"task_id": "load_customers", "table": "customers", "source_type": "parquet"},
		map[string]any{"task_id": "load_orders", "table": "orders", "source_type": "json"},
	}

	tests := []struct {
		name   string
		inputs string
		vars   map[string]*variable.Variable
		// wantJSON is compared semantically against the resulting inputs string.
		// Empty means inputs must be left unchanged.
		wantJSON string
	}{
		{
			name:   "complex variable array of objects is serialized",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
			wantJSON: `[{"source_type":"parquet","table":"customers","task_id":"load_customers"},{"source_type":"json","table":"orders","task_id":"load_orders"}]`,
		},
		{
			name:   "long variable reference form is serialized",
			inputs: "${variables.table_list.value}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
			wantJSON: `[{"source_type":"parquet","table":"customers","task_id":"load_customers"},{"source_type":"json","table":"orders","task_id":"load_orders"}]`,
		},
		{
			name:   "empty array is serialized",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{}},
			},
			wantJSON: `[]`,
		},
		{
			name:   "escaping and nested values are serialized",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{
					map[string]any{
						"task_id": `load "quoted" \\ ünïcodé`,
						"nested":  map[string]any{"list": []any{float64(1), true, nil, "x\ny"}},
					},
				}},
			},
			wantJSON: `[{"nested":{"list":[1,true,null,"x\ny"]},"task_id":"load \"quoted\" \\\\ ünïcodé"}]`,
		},
		{
			name:   "existing JSON string is left unchanged",
			inputs: `[{"task_id":"load_customers"}]`,
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
		},
		{
			name:   "scalar variable reference is left for normal resolution",
			inputs: "${var.table_name}",
			vars: map[string]*variable.Variable{
				"table_name": {Value: "customers"},
			},
		},
		{
			name:   "non-variable reference is left unchanged",
			inputs: "${bundle.name}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
		},
		{
			name:   "interpolated reference is left unchanged",
			inputs: `prefix-${var.table_list}`,
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
		},
		{
			name:   "missing variable is left for normal resolution",
			inputs: "${var.does_not_exist}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
		},
		{
			name:   "complex value with nested reference is left unchanged",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{
					map[string]any{"task_id": "${resources.jobs.other.name}"},
				}},
			},
		},
		{
			name:   "subpath into complex variable is serialized",
			inputs: "${var.wrapped.items}",
			vars: map[string]*variable.Variable{
				"wrapped": {Type: variable.VariableTypeComplex, Value: map[string]any{"items": complexValue}},
			},
			wantJSON: `[{"source_type":"parquet","table":"customers","task_id":"load_customers"},{"source_type":"json","table":"orders","task_id":"load_orders"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := forEachTestBundle(tt.inputs, tt.vars)

			diags := bundle.Apply(t.Context(), b, resourcemutator.SerializeForEachTaskInputs())
			require.NoError(t, diags.Error())

			got := forEachInputs(t, b)
			if tt.wantJSON == "" {
				assert.Equal(t, tt.inputs, got)
				return
			}
			assert.JSONEq(t, tt.wantJSON, got)
		})
	}
}

func TestSerializeForEachTaskInputsNestedForEach(t *testing.T) {
	b := forEachTestBundle("${var.outer}", map[string]*variable.Variable{
		"outer": {Type: variable.VariableTypeComplex, Value: []any{map[string]any{"a": "b"}}},
		"inner": {Type: variable.VariableTypeComplex, Value: []any{map[string]any{"c": "d"}}},
	})
	// Nest a second for_each_task inside the nested task.
	outer := &b.Config.Resources.Jobs["my_job"].Tasks[0]
	outer.ForEachTask.Task = jobs.Task{
		TaskKey:     "middle",
		ForEachTask: &jobs.ForEachTask{Inputs: "${var.inner}"},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.SerializeForEachTaskInputs())
	require.NoError(t, diags.Error())

	assert.JSONEq(t, `[{"a":"b"}]`, forEachInputs(t, b))
	assert.JSONEq(t, `[{"c":"d"}]`, b.Config.Resources.Jobs["my_job"].Tasks[0].ForEachTask.Task.ForEachTask.Inputs)
}

// Complex variables substituted into unrelated string fields must keep failing
// validation as before; the serialization only applies to for_each_task.inputs.
func TestSerializeForEachTaskInputsPreservesUnrelatedValidation(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{
					map[string]any{"task_id": "load_customers"},
				}},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						JobSettings: jobs.JobSettings{
							Name: "${var.table_list}",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.SerializeForEachTaskInputs())
	require.NoError(t, diags.Error())
	assert.Equal(t, "${var.table_list}", b.Config.Resources.Jobs["my_job"].Name)

	diags = bundle.Apply(t.Context(), b, mutator.ResolveVariableReferencesOnlyResources())
	summaries := make([]string, 0, len(diags))
	for _, d := range diags {
		summaries = append(summaries, d.Summary)
	}
	assert.Contains(t, summaries, "expected string, found sequence")
}
