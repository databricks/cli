package flags

import (
	"testing"

	"github.com/databricks/bricks/libs/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevelFlagDefault(t *testing.T) {
	f := NewLogLevelFlag()
	assert.Equal(t, logger.LevelDisabled, f.Level())
	assert.Equal(t, "disabled", f.String())
}

func TestLogLevelFlagSetValid(t *testing.T) {
	f := NewLogLevelFlag()
	err := f.Set("info")
	require.NoError(t, err)
	assert.Equal(t, logger.LevelInfo, f.Level())
	assert.Equal(t, "info", f.String())
}

func TestLogLevelFlagSetInvalid(t *testing.T) {
	f := NewLogLevelFlag()
	err := f.Set("invalid")
	assert.ErrorContains(t, err, "accepted arguments are ")
}
