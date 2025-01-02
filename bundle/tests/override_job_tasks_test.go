package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideTasksDev(t *testing.T) {
	b := loadTarget(t, "./override_job_tasks", "development")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 2)

	tasks := b.Config.Resources.Jobs["foo"].Tasks
	assert.Equal(t, "key1", tasks[0].TaskKey)
	assert.Equal(t, "i3.xlarge", tasks[0].NewCluster.NodeTypeId)
	assert.Equal(t, 1, tasks[0].NewCluster.NumWorkers)
	assert.Equal(t, "./test1.py", tasks[0].SparkPythonTask.PythonFile)

	assert.Equal(t, "key2", tasks[1].TaskKey)
	assert.Equal(t, "13.3.x-scala2.12", tasks[1].NewCluster.SparkVersion)
	assert.Equal(t, "./test2.py", tasks[1].SparkPythonTask.PythonFile)
}

func TestOverrideTasksStaging(t *testing.T) {
	b := loadTarget(t, "./override_job_tasks", "staging")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].Tasks, 2)

	tasks := b.Config.Resources.Jobs["foo"].Tasks
	assert.Equal(t, "key1", tasks[0].TaskKey)
	assert.Equal(t, "13.3.x-scala2.12", tasks[0].NewCluster.SparkVersion)
	assert.Equal(t, "./test1.py", tasks[0].SparkPythonTask.PythonFile)

	assert.Equal(t, "key2", tasks[1].TaskKey)
	assert.Equal(t, "i3.2xlarge", tasks[1].NewCluster.NodeTypeId)
	assert.Equal(t, 4, tasks[1].NewCluster.NumWorkers)
	assert.Equal(t, "./test3.py", tasks[1].SparkPythonTask.PythonFile)
}
