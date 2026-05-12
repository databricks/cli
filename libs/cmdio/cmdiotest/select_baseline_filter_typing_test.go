package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterTyping pins the current promptui-driven Select
// behavior when the user types letters that filter the list. cmdio.Select
// uses StartInSearchMode: true with a case-insensitive substring searcher on
// Name, so each keystroke immediately narrows the visible options.
//
// This test exists so the upcoming bubbletea replacement can be checked
// against a known-good baseline.
func TestSelectBaseline_FilterTyping(t *testing.T) {
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

	tm.Type("a")
	tm.Golden("02-after-a")

	tm.Type("l")
	tm.Golden("03-after-al")

	tm.Type("p")
	tm.Golden("04-after-alp")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", id, "snapshot:\n%s", tm.Snapshot())
}
