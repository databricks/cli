package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlC pins the current promptui-driven Select behavior
// when the user cancels the prompt with Ctrl+C without making a selection.
// Captured as a migration baseline for the upcoming bubbletea replacement.
func TestSelectBaseline_CtrlC(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyCtrlC)

	id, err := tm.Result()
	require.Error(t, err)
	t.Logf("error: %v", err)
	t.Logf("id: %q", id)
}
