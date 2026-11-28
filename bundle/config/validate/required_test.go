package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredSkipsPureReference(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    viaref:
      schedule: ${resources.jobs.base.schedule}
    missing:
      schedule:
        timezone_id: UTC
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{Config: *root}

	diags = Required().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, `required field "quartz_cron_expression" is not set`, diags[0].Summary)
	assert.Equal(t, "resources.jobs.missing.schedule", diags[0].Paths[0].String())
}

func TestRequiredAppEnvSource(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{
			name: "value",
			env:  `{name: A, value: x}`,
		},
		{
			name: "value_from",
			env:  `{name: A, value_from: secret}`,
		},
		{
			name: "unresolved reference",
			env:  `{name: A, value: "${resources.apps.other.name}"}`,
		},
		{
			name: "whole-entry reference",
			env:  `"${resources.apps.other.config.env[0]}"`,
		},
		{
			name: "empty value",
			env:  `{name: A, value: ""}`,
			want: []string{"resources.apps.myapp.config.env[0]"},
		},
		{
			name: "no source",
			env:  `{name: A}`,
			want: []string{"resources.apps.myapp.config.env[0]"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  apps:
    myapp:
      name: myapp
      config:
        env:
          - `+tc.env+`
`))
			require.Empty(t, diags)
			b := &bundle.Bundle{Config: *root}

			diags = Required().Apply(t.Context(), b)
			var paths []string
			for _, d := range diags {
				assert.Equal(t, diag.Error, d.Severity)
				assert.Equal(t, `app environment variable "A" must set value or value_from`, d.Summary)
				assert.NotEmpty(t, d.Locations)
				paths = append(paths, d.Paths[0].String())
			}
			assert.Equal(t, tc.want, paths)
		})
	}
}

func TestRequiredSkipsNullField(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    j:
      schedule: null
      tasks:
        - task_key: a
          notebook_task:
      webhook_notifications:
        on_start:
          -
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{Config: *root}

	// A null sequence item is still a (zero-valued) element, so it is checked.
	diags = Required().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, `required field "id" is not set`, diags[0].Summary)
	assert.Equal(t, "resources.jobs.j.webhook_notifications.on_start[0]", diags[0].Paths[0].String())
}
