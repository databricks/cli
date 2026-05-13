package cmdiotest_test

import (
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_DeleteKeyExits pins a surprising behavior of
// [cmdio.RunPrompt]: pressing the Delete key (\x1b[3~) exits the prompt with
// io.EOF, just like Ctrl+D would on an empty line.
//
// Why: chzyer/readline maps both Delete and Ctrl+D to the same internal rune
// (CharDelete = 4). Its CharDelete handler treats a non-empty buffer as
// "forward-delete" and an empty buffer as EOF. Promptui's listener resets
// readline's buffer to empty after every keystroke (cur is the source of
// truth, not o.buf), so from readline's perspective the buffer is always
// empty when Delete arrives — every Delete press takes the EOF path.
//
// This test pins the current behavior so that any change (e.g. a promptui or
// readline upgrade that splits the two keys) is intentional.
func TestPromptBaseline_DeleteKeyExits(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")

	// Type some content first to prove the buffer is non-empty from the user's
	// perspective. This is what makes the behavior surprising: the prompt
	// still exits even though the user has typed input.
	tm.Type("hello")
	tm.Type(termtest.KeyDelete)

	v, err := tm.Result()
	require.Error(t, err, "raw output: %q", tm.Raw())
	assert.ErrorIs(t, err, io.EOF)
	assert.Empty(t, v, "Delete-as-EOF discards typed input")
}
