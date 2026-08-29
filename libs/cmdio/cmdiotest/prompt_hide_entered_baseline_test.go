package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_HideEnteredFalse pins the default post-Enter rendering
// of [cmdio.RunPrompt]: with HideEntered=false (the default), the entered
// value is shown alongside the label after the prompt closes.
func TestPromptBaseline_HideEnteredFalse(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label:       "Workspace name",
		HideEntered: false,
	})
	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", v, "snapshot:\n%s", tm.Snapshot())

	tm.Golden("01-after-enter")
}

// TestPromptBaseline_HideEnteredTrue pins that HideEntered=true clears the
// prompt frame after the user submits, leaving no trace of the entered value
// on screen. This is the path used by [cmdio.Secret].
func TestPromptBaseline_HideEnteredTrue(t *testing.T) {
	t.Parallel()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{
		Label:       "Workspace name",
		HideEntered: true,
	})
	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", v, "snapshot:\n%s", tm.Snapshot())

	tm.Golden("01-after-enter")
}
