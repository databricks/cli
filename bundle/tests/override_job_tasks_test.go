package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideJobTasksDev(t *testing.T) {
	b := loadTarget(t, "./override_job_tasks", "development")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 2)

	tasks := b.Config.Resources.Jobs["foo"].Tasks
	assert.Equal(t, tasks[0].TaskKey, "key1")
	assert.Equal(t, tasks[0].NewCluster.NodeTypeId, "i3.xlarge")
	assert.Equal(t, tasks[0].NewCluster.NumWorkers, 1)
	assert.Equal(t, tasks[0].SparkPythonTask.PythonFile, "./test1.py")

	assert.Equal(t, tasks[1].TaskKey, "key2")
	assert.Equal(t, tasks[1].NewCluster.SparkVersion, "13.3.x-scala2.12")
	assert.Equal(t, tasks[1].SparkPythonTask.PythonFile, "./test2.py")
}

func TestOverrideJobTasksStaging(t *testing.T) {
	b := loadTarget(t, "./override_job_tasks", "staging")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 2)

	tasks := b.Config.Resources.Jobs["foo"].Tasks
	assert.Equal(t, tasks[0].TaskKey, "key1")
	assert.Equal(t, tasks[0].NewCluster.SparkVersion, "13.3.x-scala2.12")
	assert.Equal(t, tasks[0].SparkPythonTask.PythonFile, "./test1.py")

	assert.Equal(t, tasks[1].TaskKey, "key2")
	assert.Equal(t, tasks[1].NewCluster.NodeTypeId, "i3.2xlarge")
	assert.Equal(t, tasks[1].NewCluster.NumWorkers, 4)
	assert.Equal(t, tasks[1].SparkPythonTask.PythonFile, "./test3.py")
}
