package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobAndPipelineDevelopment(t *testing.T) {
	b := loadTarget(t, "./job_and_pipeline", "development")
	assert.Empty(t, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.Equal(t, config.Development, b.Config.Bundle.Mode)
	assert.True(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_development", p.Target)
}

func TestJobAndPipelineStaging(t *testing.T) {
	b := loadTarget(t, "./job_and_pipeline", "staging")
	assert.Empty(t, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.False(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_staging", p.Target)
}

func TestJobAndPipelineProduction(t *testing.T) {
	b := loadTarget(t, "./job_and_pipeline", "production")
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Len(t, b.Config.Resources.Pipelines, 1)

	p := b.Config.Resources.Pipelines["nyc_taxi_pipeline"]
	assert.False(t, p.Development)
	require.Len(t, p.Libraries, 1)
	assert.Equal(t, "./dlt/nyc_taxi_loader", p.Libraries[0].Notebook.Path)
	assert.Equal(t, "nyc_taxi_production", p.Target)

	j := b.Config.Resources.Jobs["pipeline_schedule"]
	assert.Equal(t, "Daily refresh of production pipeline", j.Name)
	require.Len(t, j.Tasks, 1)
	assert.NotEmpty(t, j.Tasks[0].PipelineTask.PipelineId)
}
