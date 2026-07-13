package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_CtrlH pins that Ctrl+H deletes the character to the
// left of the cursor in [cmdio.RunPrompt] — the same as the Backspace key.
// Ctrl+H sends BS (0x08) and Backspace sends DEL (0x7f); the prompt model
// handles both as backspace, making the control-character form a de-facto
// alias. This test pins that equivalence so a future change can't silently
// drop it.
func TestPromptBaseline_CtrlH(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Golden("01-typed-hello")

	tm.Type(termtest.KeyCtrlH)
	tm.Type(termtest.KeyCtrlH)
	tm.Golden("02-after-ctrl-h-twice")

	tm.Type(termtest.KeyEnter)
	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hel", v)
}
