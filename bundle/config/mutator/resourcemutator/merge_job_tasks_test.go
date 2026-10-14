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

func TestMergeJobTasks(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("foo"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("13.3.x-scala2.12"),
										NodeTypeId:   new("i3.xlarge"),
										NumWorkers:   new(2),
									},
									Libraries: []jobs.Library{
										{Whl: new("package1")},
									},
								},
								{
									TaskKey: new("bar"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										SparkVersion: new("10.4.x-scala2.12"),
									},
								},
								{
									TaskKey: new("foo"),
									NewCluster: &jobs.ClusterSpec_NewCluster{
										NodeTypeId: new("i3.2xlarge"),
										NumWorkers: new(4),
									},
									Libraries: []jobs.Library{
										{Pypi: &jobs.PythonPyPiLibrary{
											Package: new("package2"),
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

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeJobTasks())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["foo"]

	assert.Len(t, j.Tasks, 2)
	assert.Equal(t, "foo", utils.Deref(j.Tasks[1].TaskKey))
	assert.Equal(t, "bar", utils.Deref(j.Tasks[0].TaskKey))

	// This task was merged with a subsequent one.
	task0 := j.Tasks[1]
	cluster := task0.NewCluster
	assert.Equal(t, "13.3.x-scala2.12", utils.Deref(cluster.SparkVersion))
	assert.Equal(t, "i3.2xlarge", utils.Deref(cluster.NodeTypeId))
	assert.Equal(t, 4, utils.Deref(cluster.NumWorkers))
	assert.Len(t, task0.Libraries, 2)
	assert.Equal(t, "package1", utils.Deref(task0.Libraries[0].Whl))
	assert.Equal(t, "package2", utils.Deref(task0.Libraries[1].Pypi.Package))

	// This task was left untouched.
	task1 := j.Tasks[0].NewCluster
	assert.Equal(t, "10.4.x-scala2.12", utils.Deref(task1.SparkVersion))
}

func TestMergeJobTasksWithNilKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
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

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeJobTasks())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 1)
}
