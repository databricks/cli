package cmdiotest_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_ArrowPageNav pins that the right and left arrow
// keys page through the selection list — the same as Ctrl+F / Ctrl+B
// (covered by TestSelectBaseline_CtrlFCtrlB). Promptui maps both pairs
// to KeyForward / KeyBackward, which the select widget treats as
// page-down / page-up rather than item-by-item movement.
func TestSelectBaseline_ArrowPageNav(t *testing.T) {
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

	tm.Type(termtest.KeyRight)
	tm.Golden("02-after-right")

	tm.Type(termtest.KeyRight)
	tm.Golden("03-after-right-twice")

	tm.Type(termtest.KeyLeft)
	tm.Golden("04-after-left")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "id03", id, "snapshot:\n%s", tm.Snapshot())
}
