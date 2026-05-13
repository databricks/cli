package cmdiotest_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_Scroll pins the current promptui scrolling behavior for a
// list larger than promptui's default visible window. It feeds enough KeyDown
// presses to reach the last item and then keeps pressing past it, so the
// goldens capture both the bottom-of-list state and the past-bottom state.
//
// This baseline lets the upcoming bubbletea reimplementation be diffed against
// the exact rendering promptui produces today.
func TestSelectBaseline_Scroll(t *testing.T) {
	t.Parallel()
	items := make([]cmdio.Tuple, 0, 12)
	for i := 1; i <= 12; i++ {
		items = append(items, cmdio.Tuple{
			Name: fmt.Sprintf("item-%02d", i),
			Id:   fmt.Sprintf("id%02d", i),
		})
	}

	tm := termtest.NewSelectOrdered(t, items, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("item-01")
	tm.Golden("01-initial")

	for range 11 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("02-bottom")

	for range 5 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("03-past-bottom")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "id12", id, "snapshot:\n%s", tm.Snapshot())
}
