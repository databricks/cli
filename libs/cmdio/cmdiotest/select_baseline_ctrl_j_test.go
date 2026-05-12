package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlJ pins that Ctrl+J submits the Select prompt
// cleanly. Ctrl+J sends LF (0x0a) and Enter sends CR (0x0d); chzyer/readline
// maps both to CharEnter, so Ctrl+J ends the read loop the same way Enter
// does. The test does not assert which item is returned: promptui's
// listener has a bug where Ctrl+J resets the highlight to the first item
// before returning (Enter from the same state correctly returns "b" —
// pinned by TestSelectBaseline_DownEnter), and a future implementation
// is free to fix that. We only require that submission succeeds and that
// the returned id is one of the valid items.
func TestSelectBaseline_CtrlJ(t *testing.T) {
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
	assert.Contains(t, []string{"a", "b", "g"}, id, "snapshot:\n%s", tm.Snapshot())
}
