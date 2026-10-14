package resourcemutator_test

import (
	"github.com/databricks/cli/libs/utils"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
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
									JobClusterKey: new("foo"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.3.x-scala2.12"),
										NodeTypeId:   new("i3.xlarge"),
										NumWorkers:   new(2),
									},
								},
								{
									JobClusterKey: new("bar"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("10.4.x-scala2.12"),
									},
								},
								{
									JobClusterKey: new("foo"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										NodeTypeId: new("i3.2xlarge"),
										NumWorkers: new(4),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeJobClusters())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["foo"]

	assert.Len(t, j.JobClusters, 2)
	assert.Equal(t, "foo", utils.Deref(j.JobClusters[0].JobClusterKey))
	assert.Equal(t, "bar", utils.Deref(j.JobClusters[1].JobClusterKey))

	// This job cluster was merged with a subsequent one.
	jc0 := j.JobClusters[0].NewCluster
	assert.Equal(t, "13.3.x-scala2.12", utils.Deref(jc0.SparkVersion))
	assert.Equal(t, "i3.2xlarge", utils.Deref(jc0.NodeTypeId))
	assert.Equal(t, 4, utils.Deref(jc0.NumWorkers))

	// This job cluster was left untouched.
	jc1 := j.JobClusters[1].NewCluster
	assert.Equal(t, "10.4.x-scala2.12", utils.Deref(jc1.SparkVersion))
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
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.3.x-scala2.12"),
										NodeTypeId:   new("i3.xlarge"),
										NumWorkers:   new(2),
									},
								},
								{
									NewCluster: &jobs.ClusterSpec_NewCluster{
										NodeTypeId: new("i3.2xlarge"),
										NumWorkers: new(4),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeJobClusters())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs["foo"].JobClusters, 1)
}
