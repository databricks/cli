package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusters(t *testing.T) {
	b := load(t, "./clusters")
	assert.Equal(t, "clusters", b.Config.Bundle.Name)

	cluster := b.Config.Resources.Clusters["foo"]
	assert.Equal(t, "foo", cluster.ClusterName)
	assert.Equal(t, "13.3.x-scala2.12", cluster.SparkVersion)
	assert.Equal(t, "i3.xlarge", cluster.NodeTypeId)
	assert.Equal(t, 2, cluster.NumWorkers)
	assert.Equal(t, "2g", cluster.SparkConf["spark.executor.memory"])
	assert.Equal(t, 2, cluster.Autoscale.MinWorkers)
	assert.Equal(t, 7, cluster.Autoscale.MaxWorkers)
}

func TestClustersOverride(t *testing.T) {
	b := loadTarget(t, "./clusters", "development")
	assert.Equal(t, "clusters", b.Config.Bundle.Name)

	cluster := b.Config.Resources.Clusters["foo"]
	assert.Equal(t, "foo-override", cluster.ClusterName)
	assert.Equal(t, "15.2.x-scala2.12", cluster.SparkVersion)
	assert.Equal(t, "m5.xlarge", cluster.NodeTypeId)
	assert.Equal(t, 3, cluster.NumWorkers)
	assert.Equal(t, "4g", cluster.SparkConf["spark.executor.memory"])
	assert.Equal(t, "4g", cluster.SparkConf["spark.executor.memory2"])
	assert.Equal(t, 1, cluster.Autoscale.MinWorkers)
	assert.Equal(t, 3, cluster.Autoscale.MaxWorkers)
}
