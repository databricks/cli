package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlCWithFilter pins the cancel path when the search
// filter is non-empty. Readline interprets Ctrl+C globally as interrupt; a
// naive replacement could rebind it to "clear input" first and only cancel on
// the second press. The error sentinel and an empty returned id must match
// the no-filter case.
func TestSelectBaseline_CtrlCWithFilter(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")

	tm.Type("xyz")
	tm.Golden("01-no-results-with-filter")

	tm.Type(termtest.KeyCtrlC)

	id, err := tm.Result()
	require.Error(t, err)
	assert.EqualError(t, err, "^C")
	assert.Empty(t, id)
}
