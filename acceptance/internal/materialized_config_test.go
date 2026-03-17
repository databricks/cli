package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMaterializedConfigIncludesPhase(t *testing.T) {
	local := true
	cloud := false

	config := TestConfig{
		Local: localPtr(local),
		Cloud: localPtr(cloud),
		Phase: 1,
	}

	out, err := GenerateMaterializedConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "Local = true\nCloud = false\nPhase = 1\n", out)
}

func TestGenerateMaterializedConfigOmitsDefaultPhase(t *testing.T) {
	local := true
	cloud := false

	config := TestConfig{
		Local: localPtr(local),
		Cloud: localPtr(cloud),
	}

	out, err := GenerateMaterializedConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "Local = true\nCloud = false\n", out)
}

func localPtr[T any](v T) *T {
	return &v
}
