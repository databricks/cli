package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJobResource(t *testing.T) {
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

	// Confirm that this example matches the schema of the job resource.
	{
		var job resources.Job
		_, diags := convert.Normalize(&job, v)
		assert.Len(t, diags, 1)
		assert.Contains(t, diags[0].Summary, "unknown field: foobar")
	}

	// Convert the dyn.Value to a Terraform job resource.
	nv, err := convertJobResource(context.Background(), v)
	require.NoError(t, err)

	// Confirm that the conversion matches the schema of the Terraform resource.
	{
		var job schema.ResourceJob
		err := convert.ToTyped(&job, nv)
		require.NoError(t, err)
		assert.Equal(t, schema.ResourceJob{
			Name:              "job_name",
			Continuous:        &schema.ResourceJobContinuous{PauseStatus: "UNPAUSED"},
			MaxConcurrentRuns: 0,
			EmailNotifications: &schema.ResourceJobEmailNotifications{
				NoAlertForSkippedRuns: false,
				OnFailure:             []string{"jane@doe.com"},
				OnStart:               []string{"jane@doe.com"},
				OnSuccess:             []string{"jane@doe.com"},
			},
			GitSource: &schema.ResourceJobGitSource{
				Branch:   "branch",
				Commit:   "commit",
				Provider: "provider",
				Tag:      "tag",
				Url:      "url",
			},
			Tags: map[string]string{
				"tag1":  "value1",
				"tag2":  "value2",
				"empty": "",
			},
			JobCluster: []schema.ResourceJobJobCluster{
				{
					JobClusterKey: "job_cluster_key",
					NewCluster: &schema.ResourceJobJobClusterNewCluster{
						NumWorkers:   0,
						SparkVersion: "14.3.x-scala2.12",
					},
				},
			},
			Task: []schema.ResourceJobTask{
				{
					TaskKey:       "task_key",
					JobClusterKey: "job_cluster_key",
					NotebookTask: &schema.ResourceJobTaskNotebookTask{
						BaseParameters: map[string]string{
							"param1": "value1",
							"param2": "value2",
						},
						NotebookPath: "path",
					},
				},
			},
		}, job)
	}
}
