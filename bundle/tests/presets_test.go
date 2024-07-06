package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestPresetsDev(t *testing.T) {
	b := loadTarget(t, "./presets", "dev")

	assert.Equal(t, "myprefix", b.Config.Presets.Prefix)
	assert.Equal(t, config.Paused, b.Config.Presets.TriggerPauseStatus)
	assert.Equal(t, 10, b.Config.Presets.JobsMaxConcurrentRuns)
	assert.Equal(t, true, *b.Config.Presets.PipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Presets.Tags)["dev"])
	assert.NotContains(t, b.Config.Presets.Tags, "prod")
	assert.NotContains(t, b.Config.Presets.Tags, "~prod")

	// Tags transform is overridden by the dev target, so prod is not set
	assert.Equal(t, "", (*b.Config.Presets.Tags)["prod"])
}

func TestPresetsProd(t *testing.T) {
	b := loadTarget(t, "./presets", "prod")

	assert.Equal(t, false, *b.Config.Presets.PipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Presets.Tags)["prod"])
}
