package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_CtrlJ pins that Ctrl+J submits the prompt in
// [cmdio.RunPrompt] — the same as the Enter (Return) key. Enter sends CR
// (0x0d) and Ctrl+J sends LF (0x0a); the prompt model treats both as
// submit. A future change that only reacts to CR would silently swallow
// Ctrl+J; this test pins the parity.
func TestPromptBaseline_CtrlJ(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label: "Workspace name",
	})
	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Golden("01-typed-hello")

	tm.Type(termtest.KeyCtrlJ)
	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", v)
}
