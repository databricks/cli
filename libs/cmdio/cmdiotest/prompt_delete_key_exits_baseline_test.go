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
// io.EOF, just like Ctrl+D would on an empty line — and discards any input
// the user had already typed. The prompt model collapses both keys into the
// same EOF path; see prompt.go for the rationale. Pinning the behavior here
// makes sure a future change that splits the two keys is intentional.
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
