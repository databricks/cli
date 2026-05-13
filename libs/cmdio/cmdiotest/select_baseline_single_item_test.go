package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_SingleItem pins the current promptui-driven Select
// behavior when the input list contains exactly one entry. It is a migration
// baseline for the bubbletea replacement: we want to know whether promptui
// renders a prompt at all, what KeyDown does, and what id Enter returns.
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
