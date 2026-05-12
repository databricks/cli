package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_CtrlC pins Ctrl+C cancellation for RunPrompt. Mirrors
// the equivalent Secret test: error is returned, value is empty, snapshot
// captures any "^C" that the terminal echoed.
func TestPromptBaseline_CtrlC(t *testing.T) {
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")
	tm.Golden("01-empty")

	tm.Type("partial input")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyCtrlC)

	v, err := tm.Result()
	require.Error(t, err)
	assert.Empty(t, v)
	t.Logf("error: %v", err)
}
