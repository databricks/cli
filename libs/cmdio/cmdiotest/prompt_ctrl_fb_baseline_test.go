package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_CtrlFCtrlB pins that Ctrl+F and Ctrl+B move the cursor
// one character forward and backward in [cmdio.RunPrompt], the same as the
// right and left arrow keys.
//
// chzyer/readline maps Ctrl+F to CharForward and Ctrl+B to CharBackward —
// the same runes the arrow keys decode to — and promptui's Cursor.Listen
// dispatches both via its KeyForward / KeyBackward cases. So the emacs-
// style bindings are de-facto aliases for the arrow keys; this test pins
// that equivalence.
func TestPromptBaseline_CtrlFCtrlB(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Golden("01-cursor-end")

	tm.Type(termtest.KeyCtrlB)
	tm.Type(termtest.KeyCtrlB)
	tm.Golden("02-after-ctrl-b-twice")

	tm.Type(termtest.KeyCtrlF)
	tm.Golden("03-after-ctrl-f")

	tm.Type(termtest.KeyEnter)
	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", v)
}
