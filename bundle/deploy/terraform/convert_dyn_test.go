package terraform

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/require"
)

func TestConvertFromDyn(t *testing.T) {
	v := dyn.V(map[string]dyn.Value{
		"name": dyn.V("job_name"),
		"continuous": dyn.V(map[string]dyn.Value{
			"pause_status": dyn.V("UNPAUSED"),
		}),
		"max_concurrent_runs": dyn.V(0),
		"email_notifications": dyn.V(map[string]dyn.Value{
			"no_alert_for_skipped_runs": dyn.V(false),
			"on_failure":                dyn.V([]dyn.Value{dyn.V("jane@doe.com")}),
			"on_start":                  dyn.V([]dyn.Value{dyn.V("jane@doe.com")}),
			"on_success":                dyn.V([]dyn.Value{dyn.V("jane@doe.com")}),
		}),
		"git_source": dyn.V(map[string]dyn.Value{
			"git_branch":   dyn.V("branch"),
			"git_commit":   dyn.V("commit"),
			"git_provider": dyn.V("provider"),
			"git_tag":      dyn.V("tag"),
			"git_url":      dyn.V("url"),
		}),
		"tags": dyn.V(map[string]dyn.Value{
			"tag1":  dyn.V("value1"),
			"tag2":  dyn.V("value2"),
			"empty": dyn.V(""),
		}),
		"job_clusters": dyn.V([]dyn.Value{
			dyn.V(map[string]dyn.Value{
				"job_cluster_key": dyn.V("job_cluster_key"),
				"new_cluster": dyn.V(map[string]dyn.Value{
					"num_workers":   dyn.V(0),
					"spark_version": dyn.V("14.3.x-scala2.12"),
				}),
			}),
		}),
		"tasks": dyn.V([]dyn.Value{
			dyn.V(map[string]dyn.Value{
				"task_key":        dyn.V("task_key"),
				"job_cluster_key": dyn.V("job_cluster_key"),
				"notebook_task": dyn.V(map[string]dyn.Value{
					"base_parameters": dyn.V(map[string]dyn.Value{
						"param1": dyn.V("value1"),
						"param2": dyn.V("value2"),
					}),
					"notebook_path": dyn.V("path"),
				}),
			}),
		}),
		"foobar": dyn.V("baz"),
	})

	// // Confirm that this example matches the schema of the job resource.
	// {
	// 	var job resources.Job
	// 	_, diags := convert.Normalize(&job, v)
	// 	require.Empty(t, diags)
	// }

	// Convert the dyn.Valuealue to a Terraform job resource.
	nv, err := convertJobResource(context.Background(), v)
	require.NoError(t, err)

	// Confirm that the conversion matches the schema of the Terraform resource.
	{
		var job schema.ResourceJob
		_, diags := convert.Normalize(&job, nv)
		require.Empty(t, diags)
	}

}
