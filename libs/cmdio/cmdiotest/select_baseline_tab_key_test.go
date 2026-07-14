package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_TabKey pins Select's behavior when the user presses
// Tab. Tab is a common navigation key but in search-mode Select it gets
// typed into the filter; this test records that, plus how Enter behaves
// after the filter has no matches.
func TestSelectBaseline_TabKey(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyTab)
	tm.Golden("02-after-tab")

	tm.Type(termtest.KeyTab)
	tm.Golden("03-after-second-tab")

	// Enter does not terminate the prompt: in search mode with no matching
	// items (the two Tab keystrokes typed two tab characters into the
	// filter), the model treats Enter as inert. Ctrl+C cancels cleanly.
	tm.Type(termtest.KeyEnter)
	tm.Type(termtest.KeyCtrlC)

	id, err := tm.Result()
	require.Error(t, err)
	assert.EqualError(t, err, "^C")
	assert.Empty(t, id)
}
