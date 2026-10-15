package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideComputeModeDevelopment(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode:      config.Development,
				ClusterId: "newClusterID",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								NewCluster: &jobs.ClusterSpec_NewCluster{
									SparkVersion: new("14.2.x-scala2.12"),
								},
							},
							{
								ExistingClusterId: new("cluster2"),
							},
							{
								EnvironmentKey: new("environment_key"),
							},
							{
								JobClusterKey: new("cluster_key"),
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Resources.Jobs["job1"].Tasks[0].NewCluster)
	assert.Equal(t, "newClusterID", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId))
	assert.Equal(t, "newClusterID", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId))
	assert.Equal(t, "newClusterID", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[2].ExistingClusterId))
	assert.Equal(t, "newClusterID", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[3].ExistingClusterId))

	assert.Nil(t, b.Config.Resources.Jobs["job1"].Tasks[0].NewCluster)
	assert.Empty(t, utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[2].EnvironmentKey))
	assert.Empty(t, utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[3].JobClusterKey))
}

func TestOverrideComputeModeDefaultIgnoresVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: "",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								NewCluster: &jobs.ClusterSpec_NewCluster{},
							},
							{
								ExistingClusterId: new("cluster2"),
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.Len(t, diags, 1)
	assert.Equal(t, "The DATABRICKS_CLUSTER_ID variable is set but is ignored since the current target does not use 'mode: development'", diags[0].Summary)
	assert.Equal(t, "cluster2", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId))
}

func TestOverrideComputePipelineTask(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								PipelineTask: &jobs.PipelineTask{},
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.NoError(t, diags.Error())
	assert.Empty(t, utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId))
}

func TestOverrideComputeForEachTask(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								ForEachTask: &jobs.ForEachTask{},
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Resources.Jobs["job1"].Tasks[0].ForEachTask.Task)
}

func TestOverrideComputeModeProduction(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode:      config.Production,
				ClusterId: "newClusterID",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								NewCluster: &jobs.ClusterSpec_NewCluster{},
							},
							{
								ExistingClusterId: new("cluster2"),
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.Len(t, diags, 1)
	assert.Equal(t, "Setting a cluster override for a target that uses 'mode: production' is not recommended", diags[0].Summary)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Equal(t, "newClusterID", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId))
}

func TestOverrideComputeModeProductionIgnoresVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: config.Production,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Name: new("job1"),
						Tasks: []jobs.TaskSettings{
							{
								NewCluster: &jobs.ClusterSpec_NewCluster{},
							},
							{
								ExistingClusterId: new("cluster2"),
							},
						},
					}},
				},
			},
		},
	}

	m := resourcemutator.OverrideCompute()
	diags := bundle.Apply(t.Context(), b, m)
	require.Len(t, diags, 1)
	assert.Equal(t, "The DATABRICKS_CLUSTER_ID variable is set but is ignored since the current target does not use 'mode: development'", diags[0].Summary)
	assert.Equal(t, "cluster2", utils.Deref(b.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId))
}
