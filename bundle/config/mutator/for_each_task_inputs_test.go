package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeInputsValue(t *testing.T) {
	obj := func(pairs map[string]dyn.Value) dyn.Value {
		return dyn.V(pairs)
	}

	tests := []struct {
		name string
		in   dyn.Value
		// wantJSON is compared semantically against the resulting string.
		// Empty means the value must be returned unchanged.
		wantJSON string
	}{
		{
			name: "sequence of objects is serialized",
			in: dyn.V([]dyn.Value{
				obj(map[string]dyn.Value{
					"task_id": dyn.V("load_customers"),
					"table":   dyn.V("customers"),
				}),
			}),
			wantJSON: `[{"table":"customers","task_id":"load_customers"}]`,
		},
		{
			name:     "map is serialized",
			in:       obj(map[string]dyn.Value{"table": dyn.V("customers")}),
			wantJSON: `{"table":"customers"}`,
		},
		{
			name:     "empty sequence is serialized",
			in:       dyn.V([]dyn.Value{}),
			wantJSON: `[]`,
		},
		{
			name: "escaping and nested values are serialized",
			in: dyn.V([]dyn.Value{
				obj(map[string]dyn.Value{
					"task_id": dyn.V(`load "quoted" \\ ünïcodé`),
					"nested": obj(map[string]dyn.Value{
						"list": dyn.V([]dyn.Value{dyn.V(int64(1)), dyn.V(true), dyn.NilValue, dyn.V("x\ny")}),
					}),
				}),
			}),
			wantJSON: `[{"nested":{"list":[1,true,null,"x\ny"]},"task_id":"load \"quoted\" \\\\ ünïcodé"}]`,
		},
		{
			name: "string is left unchanged",
			in:   dyn.V(`[{"task_id":"load_customers"}]`),
		},
		{
			name: "int is left unchanged",
			in:   dyn.V(int64(123)),
		},
		{
			name: "reference in value is left unchanged",
			in: dyn.V([]dyn.Value{
				obj(map[string]dyn.Value{"task_id": dyn.V("${resources.jobs.other.name}")}),
			}),
		},
		{
			name: "reference in map key is left unchanged",
			in: dyn.V([]dyn.Value{
				obj(map[string]dyn.Value{"${var.key_name}": dyn.V("payload")}),
			}),
		},
		{
			name: "escaped reference in map key is serialized verbatim",
			in: dyn.V([]dyn.Value{
				obj(map[string]dyn.Value{"$${var.key_name}": dyn.V("payload")}),
			}),
			wantJSON: `[{"$${var.key_name}":"payload"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serializeInputsValue(tt.in)
			require.NoError(t, err)
			if tt.wantJSON == "" {
				assert.Equal(t, tt.in.Kind(), got.Kind())
				assert.Equal(t, tt.in.AsAny(), got.AsAny())
				return
			}
			s, ok := got.AsString()
			require.True(t, ok)
			assert.JSONEq(t, tt.wantJSON, s)
		})
	}
}

func TestSerializeInputsValuePreservesLocations(t *testing.T) {
	locs := []diag.Location{{File: "databricks.yml", Line: 3, Column: 11}}
	in := dyn.NewValue([]dyn.Value{dyn.V("a")}, locs)

	got, err := serializeInputsValue(in)
	require.NoError(t, err)
	assert.Equal(t, locs, got.Locations())
}

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

func diagSummaries(diags diag.Diagnostics) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}

func TestSerializeForEachTaskInputsPipeline(t *testing.T) {
	complexValue := []any{
		map[string]any{"task_id": "load_customers", "table": "customers", "source_type": "parquet"},
		map[string]any{"task_id": "load_orders", "table": "orders", "source_type": "json"},
	}
	complexJSON := `[{"source_type":"parquet","table":"customers","task_id":"load_customers"},{"source_type":"json","table":"orders","task_id":"load_orders"}]`

	tests := []struct {
		name   string
		inputs string
		vars   map[string]*variable.Variable
		// want is the resulting inputs value; wantJSON compares semantically
		// instead when set. warn expects the type-mismatch warning (and a
		// dropped inputs), errContains expects a resolution error.
		want        string
		wantJSON    string
		warn        bool
		errContains string
	}{
		{
			name:   "complex variable array of objects is serialized",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
			wantJSON: complexJSON,
		},
		{
			name:   "long variable reference form is serialized",
			inputs: "${variables.table_list.value}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
			wantJSON: complexJSON,
		},
		{
			name:   "subpath into complex variable is serialized",
			inputs: "${var.wrapped.items}",
			vars: map[string]*variable.Variable{
				"wrapped": {Type: variable.VariableTypeComplex, Value: map[string]any{"items": complexValue}},
			},
			wantJSON: complexJSON,
		},
		{
			name:   "existing JSON string keeps working",
			inputs: `[{"task_id":"load_customers"}]`,
			want:   `[{"task_id":"load_customers"}]`,
		},
		{
			name:   "scalar variable resolves as before",
			inputs: "${var.table_name}",
			vars: map[string]*variable.Variable{
				"table_name": {Value: "customers"},
			},
			want: "customers",
		},
		{
			name:   "interpolated complex reference still errors",
			inputs: `prefix-${var.table_list}`,
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: complexValue},
			},
			errContains: "cannot interpolate non-primitive value",
		},
		{
			name:        "missing variable still errors",
			inputs:      "${var.does_not_exist}",
			errContains: "reference does not exist",
		},
		{
			// Complex values containing resource references are intentionally
			// not serialized (see serializeForEachTaskInputs): normalization
			// reports them exactly as it did before this feature.
			name:   "complex value with nested resource reference keeps old diagnostics",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{
					map[string]any{"task_id": "${resources.jobs.other.name}"},
				}},
			},
			warn: true,
		},
		{
			// Map keys are never resolved anywhere in bundle configuration.
			// The missing key_name variable must therefore not produce a
			// resolution error; serialization is skipped and normalization
			// warns as before.
			name:   "reference in map key is not interpolated",
			inputs: "${var.table_list}",
			vars: map[string]*variable.Variable{
				"table_list": {Type: variable.VariableTypeComplex, Value: []any{
					map[string]any{"${var.key_name}": "payload"},
				}},
			},
			warn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := forEachTestBundle(tt.inputs, tt.vars)

			diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources())

			if tt.errContains != "" {
				require.Error(t, diags.Error())
				assert.ErrorContains(t, diags.Error(), tt.errContains)
				return
			}
			require.NoError(t, diags.Error())

			summaries := diagSummaries(diags)
			if tt.warn {
				assert.Contains(t, summaries, "expected string, found sequence")
				assert.NotContains(t, summaries, "reference does not exist: ${var.key_name}")
				assert.Empty(t, forEachInputs(t, b))
				return
			}

			assert.NotContains(t, summaries, "expected string, found sequence")
			got := forEachInputs(t, b)
			if tt.wantJSON != "" {
				assert.JSONEq(t, tt.wantJSON, got)
				return
			}
			assert.Equal(t, tt.want, got)
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

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources())
	require.NoError(t, diags.Error())

	assert.JSONEq(t, `[{"a":"b"}]`, forEachInputs(t, b))
	assert.JSONEq(t, `[{"c":"d"}]`, b.Config.Resources.Jobs["my_job"].Tasks[0].ForEachTask.Task.ForEachTask.Inputs)
}

// A for_each_task.inputs reference nested inside a variable-defined task list
// (tasks: ${var.job_tasks}) only materializes once the enclosing reference is
// substituted; serialization must still happen for it.
func TestSerializeForEachTaskInputsVariableDefinedTasks(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: variable-defined-tasks

variables:
  table_list:
    type: complex
    default:
      - task_id: load_customers
        table: customers
        source_type: parquet
      - task_id: load_orders
        table: orders
        source_type: json
  job_tasks:
    type: complex
    default:
      - task_key: process_tables
        for_each_task:
          inputs: ${var.table_list}
          concurrency: 2
          task:
            task_key: load_table

resources:
  jobs:
    my_job:
      tasks: ${var.job_tasks}
`))
	require.NoError(t, diags.Error())

	b := &bundle.Bundle{Config: *root}
	diags = bundle.ApplySeq(t.Context(), b,
		SetVariables(),
		ResolveVariableReferencesOnlyResources(),
	)
	require.NoError(t, diags.Error())
	assert.NotContains(t, diagSummaries(diags), "expected string, found sequence")
	assert.JSONEq(t,
		`[{"source_type":"parquet","table":"customers","task_id":"load_customers"},{"source_type":"json","table":"orders","task_id":"load_orders"}]`,
		forEachInputs(t, b))
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

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources())
	require.NoError(t, diags.Error())
	assert.Contains(t, diagSummaries(diags), "expected string, found sequence")
}
