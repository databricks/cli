package cmdiotest_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
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
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := t.Context()
	io := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io)

	require.True(t, cmdio.IsPromptSupported(ctx), "prompt support must be detected on the pty")

	type item struct {
		Name string
		Id   string
	}
	items := []item{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "c"},
	}

	type result struct {
		idx int
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		idx, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
			Label: "Pick one",
			Items: items,
			Searcher: func(input string, idx int) bool {
				return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
			},
			Active:   `> {{ .Name }} ({{ .Id }})`,
			Inactive: `  {{ .Name }} ({{ .Id }})`,
		})
		resCh <- result{idx: idx, err: err}
	}()

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

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, 1, res.idx, "expected to land on beta; snapshot:\n%s", tm.Snapshot())
}
