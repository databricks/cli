package cmdiotest_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterScroll pins viewport behavior when a filter
// narrows a long list to a count still larger than the viewport. Combines
// FilterTyping (substring search) with Scroll (12+ items) — neither test
// alone exercises the recompute-then-scroll path.
//
// 20 items named item-01 .. item-20; the filter "item-1" matches item-01
// plus item-10..item-19 = 11 items, more than the 5-row viewport.
func TestSelectBaseline_FilterScroll(t *testing.T) {
	t.Parallel()
	items := make([]cmdio.Tuple, 0, 20)
	for i := 1; i <= 20; i++ {
		items = append(items, cmdio.Tuple{
			Name: fmt.Sprintf("item-%02d", i),
			Id:   fmt.Sprintf("id%02d", i),
		})
	}

	tm := termtest.NewSelectOrdered(t, items, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("item-01")
	tm.Golden("01-initial")

	tm.Type("item-1")
	tm.Golden("02-filtered-top")

	for range 5 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("03-filtered-mid")

	for range 10 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("04-filtered-bottom")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "id19", id, "snapshot:\n%s", tm.Snapshot())
}
