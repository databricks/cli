package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideDevelopment(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode:      config.Development,
				ComputeID: "newClusterID",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.Task{
							{
								NewCluster: &compute.ClusterSpec{},
							},
							{
								ExistingClusterId: "cluster2",
							},
							{
								ComputeKey: "compute_key",
							},
							{
								JobClusterKey: "cluster_key",
							},
						},
					}},
				},
			},
		},
	}

	m := mutator.OverrideCompute()
	err := m.Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Nil(t, b.Config.Resources.Jobs["job1"].Tasks[0].NewCluster)
	assert.Equal(t, "newClusterID", b.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId)
	assert.Equal(t, "newClusterID", b.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId)
	assert.Equal(t, "newClusterID", b.Config.Resources.Jobs["job1"].Tasks[2].ExistingClusterId)
	assert.Equal(t, "newClusterID", b.Config.Resources.Jobs["job1"].Tasks[3].ExistingClusterId)

	assert.Nil(t, b.Config.Resources.Jobs["job1"].Tasks[0].NewCluster)
	assert.Empty(t, b.Config.Resources.Jobs["job1"].Tasks[2].ComputeKey)
	assert.Empty(t, b.Config.Resources.Jobs["job1"].Tasks[3].JobClusterKey)
}

func TestOverrideDevelopmentEnv(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.Task{
							{
								NewCluster: &compute.ClusterSpec{},
							},
							{
								ExistingClusterId: "cluster2",
							},
						},
					}},
				},
			},
		},
	}

	m := mutator.OverrideCompute()
	err := m.Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, "cluster2", b.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId)
}

func TestOverridePipelineTask(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.Task{
							{
								PipelineTask: &jobs.PipelineTask{},
							},
						},
					}},
				},
			},
		},
	}

	m := mutator.OverrideCompute()
	err := m.Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Empty(t, b.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId)
}

func TestOverrideProduction(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				ComputeID: "newClusterID",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.Task{
							{
								NewCluster: &compute.ClusterSpec{},
							},
							{
								ExistingClusterId: "cluster2",
							},
						},
					}},
				},
			},
		},
	}

	m := mutator.OverrideCompute()
	err := m.Apply(context.Background(), b)
	require.Error(t, err)
}

func TestOverrideProductionEnv(t *testing.T) {
	t.Setenv("DATABRICKS_CLUSTER_ID", "newClusterId")
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.Task{
							{
								NewCluster: &compute.ClusterSpec{},
							},
							{
								ExistingClusterId: "cluster2",
							},
						},
					}},
				},
			},
		},
	}

	m := mutator.OverrideCompute()
	err := m.Apply(context.Background(), b)
	require.NoError(t, err)
}
