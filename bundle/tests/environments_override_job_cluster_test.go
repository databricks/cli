package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideJobClusterDevWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_override_job_cluster", "development")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].JobClusters, 1)

	c := b.Config.Resources.Jobs["foo"].JobClusters[0]
	assert.Equal(t, "13.3.x-scala2.12", c.NewCluster.SparkVersion)
	assert.Equal(t, "i3.xlarge", c.NewCluster.NodeTypeId)
	assert.Equal(t, 1, c.NewCluster.NumWorkers)
}

func TestOverrideJobClusterStagingWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_override_job_cluster", "staging")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)
	assert.Len(t, b.Config.Resources.Jobs["foo"].JobClusters, 1)

	c := b.Config.Resources.Jobs["foo"].JobClusters[0]
	assert.Equal(t, "13.3.x-scala2.12", c.NewCluster.SparkVersion)
	assert.Equal(t, "i3.2xlarge", c.NewCluster.NodeTypeId)
	assert.Equal(t, 4, c.NewCluster.NumWorkers)
}
