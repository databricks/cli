package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_SingleItem pins Select's behavior when the input list
// contains exactly one entry: whether a prompt renders, what KeyDown does
// (no-op), and what id Enter returns.
func TestSelectBaseline_SingleItem(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "only", Id: "o"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("only")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-after-down")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "o", id, "snapshot:\n%s", tm.Snapshot())
}
