package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformersDev(t *testing.T) {
	b := loadTarget(t, "./transformers", "dev")

	assert.Equal(t, "myprefix", b.Config.Transformers.Prefix)
	assert.Equal(t, true, *b.Config.Transformers.DefaultTriggerPauseStatus)
	assert.Equal(t, 10, b.Config.Transformers.DefaultJobsMaxConcurrentRuns)
	assert.Equal(t, true, *b.Config.Transformers.DefaultPipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Transformers.Tags)["dev"])

	// Tags transformer is overridden by the dev target, so prod is not set
	assert.Equal(t, "", (*b.Config.Transformers.Tags)["prod"])
}

func TestTransformersProd(t *testing.T) {
	b := loadTarget(t, "./transformers", "prod")

	assert.Equal(t, false, *b.Config.Transformers.DefaultPipelinesDevelopment)
	assert.Equal(t, "true", (*b.Config.Transformers.Tags)["prod"])
}
