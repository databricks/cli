package flags

import (
	"testing"

	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevelFlagDefault(t *testing.T) {
	f := NewLogLevelFlag()
	assert.Equal(t, log.LevelWarn, f.Level())
	assert.Equal(t, "warn", f.String())
}

func TestLogLevelFlagSetValid(t *testing.T) {
	f := NewLogLevelFlag()
	err := f.Set("info")
	require.NoError(t, err)
	assert.Equal(t, log.LevelInfo, f.Level())
	assert.Equal(t, "info", f.String())
}

func TestLogLevelFlagSetInvalid(t *testing.T) {
	f := NewLogLevelFlag()
	err := f.Set("invalid")
	assert.ErrorContains(t, err, "accepted arguments are ")
}
