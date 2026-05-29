package cmdiotest_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_SlashEntersSearch pins that pressing "/" toggles a
// non-search-mode select prompt into search mode. The existing filter
// tests all use cmdio.SelectOrdered (which sets StartInSearchMode=true)
// so the toggle path is never exercised. Real callers that depend on it:
// cmd/auth/resolve.go and cmd/auth/profile_picker.go set
// StartInSearchMode based on len(items) > 5, so for small lists the
// only way to filter is to press "/".
func TestSelectBaseline_SlashEntersSearch(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string
		Id   string
	}
	items := []item{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "c"},
	}

	tm := termtest.NewSelect(t, cmdio.SelectOptions{
		Label: "Pick one",
		Items: items,
		Searcher: func(input string, idx int) bool {
			return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
		},
		Active:   `> {{ .Name }} ({{ .Id }})`,
		Inactive: `  {{ .Name }} ({{ .Id }})`,
	})
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial-no-search")

	// Slash toggles into search mode: a "Search:" line appears and
	// subsequent characters become the filter query.
	tm.Type("/")
	tm.Golden("02-after-slash")

	tm.Type("b")
	tm.Golden("03-filtering-b")

	tm.Type(termtest.KeyEnter)

	idx, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, 1, idx, "expected to land on beta; snapshot:\n%s", tm.Snapshot())
}
