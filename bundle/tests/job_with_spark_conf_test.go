package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobWithSparkConf(t *testing.T) {
	b := loadTarget(t, "./job_with_spark_conf", "default")
	assert.Len(t, b.Config.Resources.Jobs, 1)

	job := b.Config.Resources.Jobs["job_with_spark_conf"]
	assert.Len(t, job.JobClusters, 1)
	assert.Equal(t, "test_cluster", job.JobClusters[0].JobClusterKey)

	// Existing behavior is such that including non-string values
	// in the spark_conf map will cause the job to fail to load.
	// This is expected to be solved once we switch to the custom YAML loader.
	tasks := job.Tasks
	assert.Len(t, tasks, 0, "see https://github.com/databricks/cli/issues/992")
}
