package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPipeline(t *testing.T) {
	src := resources.Pipeline{
		CreatePipeline: pipelines.CreatePipeline{
			Name: "my pipeline",
			// This fields is not part of TF schema yet, but once we upgrade to TF version that supports it, this test will fail because run_as
			// will be exposed which is expected and test will need to be updated.
			RunAs: &pipelines.RunAs{
				UserName: "foo@bar.com",
			},
			// We expect AllowDuplicateNames and DryRun to be ignored and not passed to the TF output.
			// This is not supported by TF now, so we don't want to expose it.
			AllowDuplicateNames: true,
			DryRun:              true,
			Libraries: []pipelines.PipelineLibrary{
				{
					Notebook: &pipelines.NotebookLibrary{
						Path: "notebook path",
					},
				},
				{
					File: &pipelines.FileLibrary{
						Path: "file path",
					},
				},
			},
			Notifications: []pipelines.Notifications{
				{
					Alerts: []string{
						"on-update-fatal-failure",
					},
					EmailRecipients: []string{
						"jane@doe.com",
					},
				},
				{
					Alerts: []string{
						"on-update-failure",
						"on-flow-failure",
					},
					EmailRecipients: []string{
						"jane@doe.com",
						"john@doe.com",
					},
				},
			},
			Clusters: []pipelines.PipelineCluster{
				{
					Label:      "default",
					NumWorkers: 1,
				},
			},
		},
		Permissions: []resources.PipelinePermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = pipelineConverter{}.Convert(ctx, "my_pipeline", vin, out)
	require.NoError(t, err)

	// Assert equality on the pipeline
	assert.Equal(t, map[string]any{
		"name": "my pipeline",
		"library": []any{
			map[string]any{
				"notebook": map[string]any{
					"path": "notebook path",
				},
			},
			map[string]any{
				"file": map[string]any{
					"path": "file path",
				},
			},
		},
		"notification": []any{
			map[string]any{
				"alerts": []any{
					"on-update-fatal-failure",
				},
				"email_recipients": []any{
					"jane@doe.com",
				},
			},
			map[string]any{
				"alerts": []any{
					"on-update-failure",
					"on-flow-failure",
				},
				"email_recipients": []any{
					"jane@doe.com",
					"john@doe.com",
				},
			},
		},
		"allow_duplicate_names": true,
		"cluster": []any{
			map[string]any{
				"label":       "default",
				"num_workers": int64(1),
			},
		},
		"run_as": map[string]any{
			"user_name": "foo@bar.com",
		},
	}, out.Pipeline["my_pipeline"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		PipelineId: "${databricks_pipeline.my_pipeline.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_VIEW",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["pipeline_my_pipeline"])
}
