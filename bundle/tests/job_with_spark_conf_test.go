package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobWithSparkConf(t *testing.T) {
	b := loadTarget(t, "./job_with_spark_conf", "default")
	assert.Len(t, b.Config.Resources.Jobs, 1)

	job := b.Config.Resources.Jobs["job_with_spark_conf"]
	assert.Len(t, job.JobClusters, 1)
	assert.Equal(t, "test_cluster", job.JobClusters[0].JobClusterKey)

	// This test exists because of https://github.com/databricks/cli/issues/992.
	// It is solved for bundles as of https://github.com/databricks/cli/pull/1098.
	require.Len(t, job.JobClusters, 1)
	cluster := job.JobClusters[0]
	assert.Equal(t, "14.2.x-scala2.12", cluster.NewCluster.SparkVersion)
	assert.Equal(t, "i3.xlarge", cluster.NewCluster.NodeTypeId)
	assert.Equal(t, 2, cluster.NewCluster.NumWorkers)
	assert.Equal(t, map[string]string{
		"spark.string": "string",
		"spark.int":    "1",
		"spark.bool":   "true",
		"spark.float":  "1.2",
	}, cluster.NewCluster.SparkConf)
}
