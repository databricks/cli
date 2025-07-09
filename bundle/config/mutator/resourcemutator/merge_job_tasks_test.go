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

func TestMergeJobTasks(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "foo",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.3.x-scala2.12",
										NodeTypeId:   "i3.xlarge",
										NumWorkers:   2,
									},
									Libraries: []compute.Library{
										{Whl: "package1"},
									},
								},
								{
									TaskKey: "bar",
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "10.4.x-scala2.12",
									},
								},
								{
									TaskKey: "foo",
									NewCluster: &compute.ClusterSpec{
										NodeTypeId: "i3.2xlarge",
										NumWorkers: 4,
									},
									Libraries: []compute.Library{
										{Pypi: &compute.PythonPyPiLibrary{
											Package: "package2",
										}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobTasks())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["foo"]

	assert.Len(t, j.Tasks, 2)
	assert.Equal(t, "foo", j.Tasks[1].TaskKey)
	assert.Equal(t, "bar", j.Tasks[0].TaskKey)

	// This task was merged with a subsequent one.
	task0 := j.Tasks[1]
	cluster := task0.NewCluster
	assert.Equal(t, "13.3.x-scala2.12", cluster.SparkVersion)
	assert.Equal(t, "i3.2xlarge", cluster.NodeTypeId)
	assert.Equal(t, 4, cluster.NumWorkers)
	assert.Len(t, task0.Libraries, 2)
	assert.Equal(t, "package1", task0.Libraries[0].Whl)
	assert.Equal(t, "package2", task0.Libraries[1].Pypi.Package)

	// This task was left untouched.
	task1 := j.Tasks[0].NewCluster
	assert.Equal(t, "10.4.x-scala2.12", task1.SparkVersion)
}

func TestMergeJobTasksWithNilKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NewCluster: &compute.ClusterSpec{
										SparkVersion: "13.3.x-scala2.12",
										NodeTypeId:   "i3.xlarge",
										NumWorkers:   2,
									},
								},
								{
									NewCluster: &compute.ClusterSpec{
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

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobTasks())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 1)
}
