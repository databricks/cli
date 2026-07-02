package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestJobClusterKeyDefined(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							JobClusters: []jobs.JobCluster{
								{JobClusterKey: "do-not-exist"},
							},
							Tasks: []jobs.Task{
								{JobClusterKey: "do-not-exist"},
							},
						},
					},
				},
			},
		},
	}

	diags := JobClusterKeyDefined().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobClusterKeyNotDefined(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							Tasks: []jobs.Task{
								{JobClusterKey: "do-not-exist"},
							},
						},
					},
				},
			},
		},
	}

	diags := JobClusterKeyDefined().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "job_cluster_key do-not-exist is not defined", diags[0].Summary)
}

func TestJobClusterKeyDefinedInForEachTask(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							JobClusters: []jobs.JobCluster{
								{JobClusterKey: "do-not-exist"},
							},
							Tasks: []jobs.Task{
								{
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{JobClusterKey: "do-not-exist"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := JobClusterKeyDefined().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobClusterKeyNotDefinedInForEachTask(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							Tasks: []jobs.Task{
								{
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{JobClusterKey: "do-not-exist"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := JobClusterKeyDefined().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "job_cluster_key do-not-exist is not defined", diags[0].Summary)
	require.Len(t, diags[0].Paths, 1)
	require.Equal(t, "resources.jobs.job1.tasks[0].for_each_task.task.job_cluster_key", diags[0].Paths[0].String())
}

func TestJobClusterKeyDefinedInDifferentJob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							Tasks: []jobs.Task{
								{JobClusterKey: "do-not-exist"},
							},
						},
					},
					"job2": {
						JobSettings: jobs.JobSettings{
							Name: "job2",
							JobClusters: []jobs.JobCluster{
								{JobClusterKey: "do-not-exist"},
							},
						},
					},
				},
			},
		},
	}

	diags := JobClusterKeyDefined().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "job_cluster_key do-not-exist is not defined", diags[0].Summary)
}
