package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestPresetsDev(t *testing.T) {
	b := loadTarget(t, "./presets", "dev")

	assert.Equal(t, "myprefix", b.Config.Presets.NamePrefix)
	assert.Equal(t, config.Paused, b.Config.Presets.TriggerPauseStatus)
	assert.Equal(t, 10, b.Config.Presets.JobsMaxConcurrentRuns)
	assert.True(t, *b.Config.Presets.PipelinesDevelopment)
	assert.Equal(t, "true", b.Config.Presets.Tags["dev"])
	assert.Equal(t, "finance", b.Config.Presets.Tags["team"])
	assert.Equal(t, "false", b.Config.Presets.Tags["prod"])
}

func TestPresetsProd(t *testing.T) {
	b := loadTarget(t, "./presets", "prod")

	assert.False(t, *b.Config.Presets.PipelinesDevelopment)
	assert.Equal(t, "finance", b.Config.Presets.Tags["team"])
	assert.Equal(t, "true", b.Config.Presets.Tags["prod"])
}
