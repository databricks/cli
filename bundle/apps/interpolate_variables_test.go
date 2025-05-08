package apps

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/require"
)

func TestAppInterpolateVariables(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app_1": {
						App: apps.App{
							Name: "my_app_1",
						},
						Config: map[string]any{
							"command": []string{"echo", "hello"},
							"env": []map[string]string{
								{"name": "JOB_ID", "value": "${databricks_job.my_job.id}"},
							},
						},
					},
					"my_app_2": {
						App: apps.App{
							Name: "my_app_2",
						},
					},
				},
				Jobs: map[string]*resources.Job{
					"my_job": {
						ID: "123",
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, InterpolateVariables())
	require.Empty(t, diags)
	require.Equal(t, []any{map[string]any{"name": "JOB_ID", "value": "123"}}, b.Config.Resources.Apps["my_app_1"].Config["env"])
	require.Nil(t, b.Config.Resources.Apps["my_app_2"].Config)
}
