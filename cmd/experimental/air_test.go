package experimental

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirCommandMovedOutOfExperimental(t *testing.T) {
	cmd := newAirCmd()
	assert.True(t, cmd.Hidden)
	assert.True(t, cmd.DisableFlagParsing)

	err := cmd.RunE(cmd, []string{"run", "--file", "run.yaml"})
	require.EqualError(t, err, `the AIR commands have moved out of experimental; use "databricks air" instead`)
}
