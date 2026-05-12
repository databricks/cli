package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlNCtrlP pins that Ctrl+N and Ctrl+P move the
// selection down and up by one item — the same as the down and up arrow
// keys. Promptui exposes these as KeyNext (= readline.CharNext) and
// KeyPrev (= readline.CharPrev), the same runes the arrow keys decode
// to in chzyer/readline; this test pins that equivalence.
func TestSelectBaseline_CtrlNCtrlP(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "c"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyCtrlN)
	tm.Type(termtest.KeyCtrlN)
	tm.Golden("02-after-ctrl-n-twice")

	tm.Type(termtest.KeyCtrlP)
	tm.Golden("03-after-ctrl-p")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "b", id, "snapshot:\n%s", tm.Snapshot())
}
