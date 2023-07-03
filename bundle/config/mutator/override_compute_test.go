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

func TestOverrideCompute(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode:    config.Development,
				Compute: "newClusterID",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Name: "job1",
						Tasks: []jobs.JobTaskSettings{
							{
								NewCluster: &compute.BaseClusterInfo{},
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
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Nil(t, bundle.Config.Resources.Jobs["job1"].Tasks[0].NewCluster)
	assert.Equal(t, "newClusterID", bundle.Config.Resources.Jobs["job1"].Tasks[0].ExistingClusterId)
	assert.Equal(t, "newClusterID", bundle.Config.Resources.Jobs["job1"].Tasks[1].ExistingClusterId)
}
