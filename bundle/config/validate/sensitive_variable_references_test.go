package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sensitiveVariablesBundle() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"secret":       {Sensitive: new(true)},
				"other_secret": {Sensitive: new(true), Default: "placeholder"},
				"plain":        {Default: "placeholder"},
			},
			Resources: config.Resources{
				Secrets: map[string]*resources.Secret{
					"s": {Secret: catalog.Secret{Name: "s", Value: "placeholder", Comment: "placeholder"}},
				},
				Jobs: map[string]*resources.Job{
					"j": {JobSettings: jobs.JobSettings{
						Name: "placeholder",
						Tasks: []jobs.Task{{
							TaskKey: "main",
							NotebookTask: &jobs.NotebookTask{
								NotebookPath:   "/notebook",
								BaseParameters: map[string]string{"token": "placeholder"},
							},
						}},
					}},
				},
			},
		},
	}
}

func TestSensitiveVariableReferences(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value string
		// Path of the expected diagnostic; empty when the reference is allowed.
		wantPath string
	}{
		{"secret value", "resources.secrets.s.value", "${var.secret}", ""},
		{"sensitive variable default", "variables.other_secret.default", "Bearer ${var.secret}", ""},
		{"long form reference in sensitive variable", "variables.other_secret.default", "${variables.secret.value}", ""},
		{"non-sensitive variable referenced in job", "resources.jobs.j.name", "${var.plain}", ""},
		{"non-sensitive variable default", "variables.plain.default", "${var.secret}", "variables.plain.default"},
		{"long form reference in non-sensitive variable", "variables.plain.default", "${variables.secret.value}", "variables.plain.default"},
		{"secret comment", "resources.secrets.s.comment", "${var.secret}", "resources.secrets.s.comment"},
		{"job name", "resources.jobs.j.name", "job ${var.secret}", "resources.jobs.j.name"},
		{"nested job field", "resources.jobs.j.tasks[0].notebook_task.base_parameters.token", "${var.secret}", "resources.jobs.j.tasks[0].notebook_task.base_parameters.token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := sensitiveVariablesBundle()
			bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, tc.path, dyn.V(tc.value))
			})

			diags := SensitiveVariableReferences().Apply(t.Context(), b)
			if tc.wantPath == "" {
				assert.Empty(t, diags)
				return
			}
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, `Sensitive variable "secret" is used in a non-sensitive field`, diags[0].Summary)
			require.Len(t, diags[0].Paths, 1)
			assert.Equal(t, tc.wantPath, diags[0].Paths[0].String())
		})
	}
}

func TestSensitiveVariableReferencesSkipsTargets(t *testing.T) {
	b := sensitiveVariablesBundle()
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "targets", dyn.V(map[string]dyn.Value{
			"dev": dyn.V(map[string]dyn.Value{
				"resources": dyn.V(map[string]dyn.Value{
					"jobs": dyn.V(map[string]dyn.Value{
						"j": dyn.V(map[string]dyn.Value{"name": dyn.V("${var.secret}")}),
					}),
				}),
			}),
		}))
	})

	assert.Empty(t, SensitiveVariableReferences().Apply(t.Context(), b))
}

func TestSensitiveVariableReferencesInfersFromSensitiveResourceField(t *testing.T) {
	b := sensitiveVariablesBundle()
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.secrets.s.value", dyn.V("${var.plain}"))
	})

	assert.Empty(t, SensitiveVariableReferences().Apply(t.Context(), b))
	assert.True(t, config.IsExplicitlyEnabled(b.Config.Variables["plain"].Sensitive))
	v, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("variables.plain.sensitive"))
	require.NoError(t, err)
	assert.Equal(t, true, v.AsAny())
}

func TestSensitiveVariableReferencesInferredVariableInNonSensitiveField(t *testing.T) {
	b := sensitiveVariablesBundle()
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		v, err := dyn.Set(v, "resources.secrets.s.value", dyn.V("${var.plain}"))
		if err != nil {
			return v, err
		}
		return dyn.Set(v, "resources.jobs.j.name", dyn.V("${var.plain}"))
	})

	diags := SensitiveVariableReferences().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, `Sensitive variable "plain" is used in a non-sensitive field`, diags[0].Summary)
	require.Len(t, diags[0].Paths, 1)
	assert.Equal(t, "resources.jobs.j.name", diags[0].Paths[0].String())
}

func TestSensitiveVariableReferencesIgnoresUndeclaredVariable(t *testing.T) {
	b := sensitiveVariablesBundle()
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.secrets.s.value", dyn.V("${var.undeclared}"))
	})

	assert.Empty(t, SensitiveVariableReferences().Apply(t.Context(), b))
}
