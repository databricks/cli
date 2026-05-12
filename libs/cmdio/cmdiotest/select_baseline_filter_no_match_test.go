package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterNoMatch pins promptui's behavior when the user
// types a filter query that matches none of the items, then backspaces it
// out and hits Enter. Captured as a baseline for the bubbletea replacement.
func TestSelectBaseline_FilterNoMatch(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type("x")
	tm.Golden("02-after-x")

	tm.Type("y")
	tm.Golden("03-after-xy")

	tm.Type("z")
	tm.Golden("04-after-xyz")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("05-after-backspaces")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", id, "snapshot:\n%s", tm.Snapshot())
}
