package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_EscKey pins the current promptui-driven Select behavior
// when the user presses Esc at various states: the initial prompt, and after
// typing into the search filter. cmdio.Select uses StartInSearchMode: true,
// so the filter is active from the start.
//
// This test exists so the upcoming bubbletea replacement can be checked
// against a known-good baseline.
func TestSelectBaseline_EscKey(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyEsc)
	tm.Golden("02-esc-from-initial")

	tm.Type("a")
	tm.Golden("03-after-typing-a")

	tm.Type(termtest.KeyEsc)
	tm.Golden("04-esc-clears-filter-or-not")

	// Esc is inert in this Select model: it neither finalizes the prompt
	// nor clears the filter. So the filter is still "a" here, which matches
	// only "alpha"; Enter submits that match.
	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", id, "snapshot:\n%s", tm.Snapshot())
}
