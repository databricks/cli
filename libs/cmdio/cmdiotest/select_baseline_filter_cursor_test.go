package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterCursor pins the current promptui-driven Select
// behavior when the user has navigated to a non-first item, then types a
// filter query. This documents how the cursor moves into and out of filter
// mode so the upcoming bubbletea replacement can be checked against a known
// baseline.
func TestSelectBaseline_FilterCursor(t *testing.T) {
	t.Parallel()
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

	tm.Type(termtest.KeyDown)
	tm.Type(termtest.KeyDown)
	tm.Golden("02-on-gamma")

	tm.Type("a")
	tm.Golden("03-after-filter-a")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("04-after-clear-filter")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", id, "snapshot:\n%s", tm.Snapshot())
}
