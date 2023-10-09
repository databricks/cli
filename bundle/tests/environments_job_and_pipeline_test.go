package config_tests

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobAndPipelineDevelopmentWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "development")
	assert.Len(t, b.Config.Resources.Jobs, 0)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.Equal(t, "environments_job_and_pipeline/databricks.yml", filepath.ToSlash(p.LocalConfigFilePath))
	assert.Equal(t, b.Config.Bundle.Mode, config.Development)
	assert.True(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_development", p.Target)
}

func TestJobAndPipelineStagingWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "staging")
	assert.Len(t, b.Config.Resources.Jobs, 0)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.Equal(t, "environments_job_and_pipeline/databricks.yml", filepath.ToSlash(p.LocalConfigFilePath))
	assert.False(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_staging", p.Target)
}

func TestJobAndPipelineProductionWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_job_and_pipeline", "production")
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.Equal(t, "environments_job_and_pipeline/databricks.yml", filepath.ToSlash(p.LocalConfigFilePath))
	assert.False(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_production", p.Target)

	j := b.Config.Resources.Jobs["pipeline_schedule"]
	assert.Equal(t, "environments_job_and_pipeline/databricks.yml", filepath.ToSlash(j.LocalConfigFilePath))
	assert.Equal(t, "Daily refresh of production pipeline", j.Name)
	require.Len(t, j.Tasks, 1)
	assert.NotEmpty(t, j.Tasks[0].PipelineTask.PipelineId)
}
