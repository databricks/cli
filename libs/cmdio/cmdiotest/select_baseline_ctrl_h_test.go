package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlH pins that Ctrl+H deletes the last character
// from the search filter in [cmdio.Select] — the same as the Backspace
// key. Ctrl+H sends BS (0x08) and Backspace sends DEL (0x7f); promptui's
// readline maps both to CharBackspace inside the search buffer, so this
// test pins the equivalence for the filter editor.
func TestSelectBaseline_CtrlH(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type("alp")
	tm.Golden("02-after-typing-alp")

	tm.Type(termtest.KeyCtrlH)
	tm.Type(termtest.KeyCtrlH)
	tm.Golden("03-after-ctrl-h-twice")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", id, "snapshot:\n%s", tm.Snapshot())
}
