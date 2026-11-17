package termtest

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

func TestScreenRows_NonPositiveUsesDefault(t *testing.T) {
	assert.Equal(t, cmdio.TestTerminalHeight, screenRows(0))
	assert.Equal(t, cmdio.TestTerminalHeight, screenRows(-1))
	assert.Equal(t, 200, screenRows(200))
}

func TestScreenRows_NegativeDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = newEmulator(cmdio.TestTerminalWidth, screenRows(-5))
	})
}
