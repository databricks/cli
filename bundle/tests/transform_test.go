package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestTransformDev(t *testing.T) {
	b := loadTarget(t, "./transform", "dev")

	assert.Equal(t, "myprefix", b.Config.Transform.Prefix)
	assert.Equal(t, config.Paused, b.Config.Transform.DefaultTriggerPauseStatus)
	assert.Equal(t, 10, b.Config.Transform.DefaultJobsMaxConcurrentRuns)
	assert.Equal(t, true, *b.Config.Transform.DefaultPipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Transform.Tags)["dev"])

	// Tags transform is overridden by the dev target, so prod is not set
	assert.Equal(t, "", (*b.Config.Transform.Tags)["prod"])
}

func TestTransformProd(t *testing.T) {
	b := loadTarget(t, "./transform", "prod")

	assert.Equal(t, false, *b.Config.Transform.DefaultPipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Transform.Tags)["prod"])
}
