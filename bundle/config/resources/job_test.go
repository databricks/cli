package resources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobMergeJobClusters(t *testing.T) {
	j := &Job{
		JobSettings: &jobs.JobSettings{
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "foo",
					NewCluster: &compute.ClusterSpec{
						SparkVersion: "13.3.x-scala2.12",
						NodeTypeId:   "i3.xlarge",
						NumWorkers:   2,
					},
				},
				{
					JobClusterKey: "bar",
					NewCluster: &compute.ClusterSpec{
						SparkVersion: "10.4.x-scala2.12",
					},
				},
				{
					JobClusterKey: "foo",
					NewCluster: &compute.ClusterSpec{
						NodeTypeId: "i3.2xlarge",
						NumWorkers: 4,
					},
				},
			},
		},
	}

	err := j.MergeJobClusters()
	require.NoError(t, err)

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

func TestJobMergeTasks(t *testing.T) {
	j := &Job{
		JobSettings: &jobs.JobSettings{
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
	}

	err := j.MergeTasks()
	require.NoError(t, err)

	assert.Len(t, j.Tasks, 2)
	assert.Equal(t, "foo", j.Tasks[0].TaskKey)
	assert.Equal(t, "bar", j.Tasks[1].TaskKey)

	// This task was merged with a subsequent one.
	task0 := j.Tasks[0]
	cluster := task0.NewCluster
	assert.Equal(t, "13.3.x-scala2.12", cluster.SparkVersion)
	assert.Equal(t, "i3.2xlarge", cluster.NodeTypeId)
	assert.Equal(t, 4, cluster.NumWorkers)
	assert.Len(t, task0.Libraries, 2)
	assert.Equal(t, task0.Libraries[0].Whl, "package1")
	assert.Equal(t, task0.Libraries[1].Pypi.Package, "package2")

	// This task was left untouched.
	task1 := j.Tasks[1].NewCluster
	assert.Equal(t, "10.4.x-scala2.12", task1.SparkVersion)
}
