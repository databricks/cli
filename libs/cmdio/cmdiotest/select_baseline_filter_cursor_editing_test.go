package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterCursorEditing pins how the search filter responds
// to cursor-editing keys: ←/→, Home/End, Delete, Ctrl+W. Promptui's readline
// supports all of these inside the search buffer; whether a hand-rolled
// bubbletea filter does is the whole point of the baseline.
func TestSelectBaseline_FilterCursorEditing(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
		{Name: "delta", Id: "d"},
		{Name: "epsilon", Id: "e"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type("alp")
	tm.Golden("02-after-typing-alp")

	tm.Type(termtest.KeyLeft)
	tm.Type(termtest.KeyLeft)
	tm.Type("X")
	tm.Golden("03-after-insert-mid")

	tm.Type(termtest.KeyHome)
	tm.Type("Y")
	tm.Golden("04-after-insert-at-start")

	tm.Type(termtest.KeyEnd)
	tm.Type("Z")
	tm.Golden("05-after-insert-at-end")

	tm.Type(termtest.KeyCtrlU)
	tm.Golden("06-after-ctrl-u")

	tm.Type("alpha")
	tm.Type(termtest.KeyCtrlW)
	tm.Golden("07-after-ctrl-w")

	tm.Type(termtest.KeyCtrlC)

	id, err := tm.Result()
	require.Error(t, err)
	t.Logf("error: %v", err)
	t.Logf("id: %q", id)
}
