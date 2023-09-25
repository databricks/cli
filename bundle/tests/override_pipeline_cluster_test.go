package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverridePipelineClusterDev(t *testing.T) {
	b := loadTarget(t, "./override_pipeline_cluster", "development")
	assert.Equal(t, "job", b.Config.Resources.Pipelines["foo"].Name)
	assert.Len(t, b.Config.Resources.Pipelines["foo"].Clusters, 1)

	c := b.Config.Resources.Pipelines["foo"].Clusters[0]
	assert.Equal(t, map[string]string{"foo": "bar"}, c.SparkConf)
	assert.Equal(t, "i3.xlarge", c.NodeTypeId)
	assert.Equal(t, 1, c.NumWorkers)
}

func TestOverridePipelineClusterStaging(t *testing.T) {
	b := loadTarget(t, "./override_pipeline_cluster", "staging")
	assert.Equal(t, "job", b.Config.Resources.Pipelines["foo"].Name)
	assert.Len(t, b.Config.Resources.Pipelines["foo"].Clusters, 1)

	c := b.Config.Resources.Pipelines["foo"].Clusters[0]
	assert.Equal(t, map[string]string{"foo": "bar"}, c.SparkConf)
	assert.Equal(t, "i3.2xlarge", c.NodeTypeId)
	assert.Equal(t, 4, c.NumWorkers)
}
