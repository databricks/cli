package config_tests

import (
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobAndPipelineDevelopmentWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "development")
	assert.Empty(t, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.Equal(t, config.Development, b.Config.Bundle.Mode)
	assert.True(t, utils.Deref(p.Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", utils.Deref(p.Libraries[0].Notebook.Path))
	assert.Equal(t, "nyc_taxi_development", utils.Deref(p.Target))
}

func TestJobAndPipelineStagingWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "staging")
	assert.Empty(t, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.False(t, utils.Deref(p.Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", utils.Deref(p.Libraries[0].Notebook.Path))
	assert.Equal(t, "nyc_taxi_staging", utils.Deref(p.Target))
}

func TestJobAndPipelineProductionWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "production")
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.False(t, utils.Deref(p.Development)) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", utils.Deref(p.Libraries[0].Notebook.Path))
	assert.Equal(t, "nyc_taxi_production", utils.Deref(p.Target))

	j := b.Config.Resources.Jobs["pipeline_schedule"]
	assert.Equal(t, "Daily refresh of production pipeline", utils.Deref(j.Name))
	require.Len(t, j.Tasks, 1)
	assert.NotEmpty(t, utils.Deref(j.Tasks[0].PipelineTask.PipelineId))
}
