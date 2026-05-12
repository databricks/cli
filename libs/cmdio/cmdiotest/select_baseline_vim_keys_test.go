package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_VimKeys pins how the current promptui-driven Select
// reacts to vim-style 'j' and 'k' keys. promptui's Select is configured with
// StartInSearchMode: true, so letters likely flow into the filter rather than
// acting as navigation. This baseline captures whatever it does today so the
// upcoming bubbletea replacement can decide deliberately whether to preserve,
// drop, or change that behavior.
func TestSelectBaseline_VimKeys(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
		{Name: "delta", Id: "d"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type("j")
	tm.Golden("02-after-j")

	tm.Type("k")
	tm.Golden("03-after-jk")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("04-after-backspaces")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	t.Logf("Enter selected id=%q (snapshot:\n%s\n)", id, tm.Snapshot())
}
