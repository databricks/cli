package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlJ pins that Ctrl+J submits the Select prompt
// cleanly. Ctrl+J sends LF (0x0a) and Enter sends CR (0x0d); the bubbletea
// model treats both as submit, so Ctrl+J ends the prompt the same way Enter
// does. After one KeyDown the highlight is on "b" and that's what gets
// returned — pin the exact value so a future change can't silently return a
// different index while still rendering the same screen.
func TestSelectBaseline_CtrlJ(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-after-down")

	tm.Type(termtest.KeyCtrlJ)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	// promptui today returns "a" here (the first item) instead of the
	// highlighted "b"; a future implementation may return "b". Accept any
	// valid id so the test pins submission, not the parity miss.
	assert.Equal(t, "b", id, "snapshot:\n%s", tm.Snapshot())
}
