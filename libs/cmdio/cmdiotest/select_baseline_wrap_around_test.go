package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_WrapAround pins promptui's behavior at the list edges:
// pressing Up on the first item and Down past the last item. This baseline
// lets the bubbletea replacement be checked against the current behavior.
func TestSelectBaseline_WrapAround(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	// Up from the top: does promptui wrap to the last item, pin to alpha,
	// or do nothing visible?
	tm.Type(termtest.KeyUp)
	tm.Golden("02-up-from-top")

	// Five Downs: with three items, this overshoots the bottom by two,
	// exposing whether Down wraps, pins, or beeps past the last item.
	for range 5 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("03-down-past-bottom")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	t.Logf("selected: %s", id)
}
