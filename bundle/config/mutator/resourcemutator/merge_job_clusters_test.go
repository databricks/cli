package resourcemutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestMergeJobClusters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									JobClusterKey: "foo",
									NewCluster: compute.ClusterSpec{
										SparkVersion: "13.3.x-scala2.12",
										NodeTypeId:   "i3.xlarge",
										NumWorkers:   2,
									},
								},
								{
									JobClusterKey: "bar",
									NewCluster: compute.ClusterSpec{
										SparkVersion: "10.4.x-scala2.12",
									},
								},
								{
									JobClusterKey: "foo",
									NewCluster: compute.ClusterSpec{
										NodeTypeId: "i3.2xlarge",
										NumWorkers: 4,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobClusters())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["foo"]

	assert.Len(t, j.JobClusters, 2)
	assert.Equal(t, "foo", j.JobClusters[0].JobClusterKey)
	assert.Equal(t, "bar", j.JobClusters[1].JobClusterKey)

	// This job cluster was merged with a subsequent one.
	jc0 := j.JobClusters[0].NewCluster
	assert.Equal(t, "13.3.x-scala2.12", jc0.SparkVersion)
	assert.Equal(t, "i3.2xlarge", jc0.NodeTypeId)
	assert.Equal(t, 4, jc0.NumWorkers)

	// This job cluster was left untouched.
	jc1 := j.JobClusters[1].NewCluster
	assert.Equal(t, "10.4.x-scala2.12", jc1.SparkVersion)
}

func TestMergeJobClustersWithNilKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							JobClusters: []jobs.JobCluster{
								{
									NewCluster: compute.ClusterSpec{
										SparkVersion: "13.3.x-scala2.12",
										NodeTypeId:   "i3.xlarge",
										NumWorkers:   2,
									},
								},
								{
									NewCluster: compute.ClusterSpec{
										NodeTypeId: "i3.2xlarge",
										NumWorkers: 4,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobClusters())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs["foo"].JobClusters, 1)
}
