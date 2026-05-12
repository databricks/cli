package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_UTF8 pins multi-byte rune handling: typing "café"
// (4 runes, 5 bytes) renders as 4 cells, one Backspace deletes one rune
// not one byte, and the returned value preserves the original code points.
//
// Promptui delegates to readline which is rune-aware. A bubbletea
// reimplementation that counts bytes will silently corrupt non-ASCII input
// even though ASCII tests still pass — exactly the kind of regression a
// migration baseline is meant to catch.
func TestPromptBaseline_UTF8(t *testing.T) {
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Name",
	})
	tm.WaitFor("Name")
	tm.Golden("01-empty")

	tm.Type("café")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type("é")
	tm.Golden("04-restored")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "café", v, "snapshot:\n%s", tm.Snapshot())
}
