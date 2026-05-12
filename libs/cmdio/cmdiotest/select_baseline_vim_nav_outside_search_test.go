package cmdiotest_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_VimNavOutsideSearch pins promptui's vim-style
// navigation when the prompt opens outside search mode. With
// StartInSearchMode=false, j/k move the highlighted item by one and
// h/l page through the list; with search mode enabled (the default in
// cmdio.SelectOrdered) those letters would flow into the filter
// instead. Real callers that hit this branch:
// cmd/auth/resolve.go and cmd/auth/profile_picker.go both set
// StartInSearchMode based on len(items) > 5, so small lists open
// outside search mode.
func TestSelectBaseline_VimNavOutsideSearch(t *testing.T) {
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
	items := make([]item, 0, 12)
	for i := 1; i <= 12; i++ {
		items = append(items, item{
			Name: fmt.Sprintf("item-%02d", i),
			Id:   fmt.Sprintf("id%02d", i),
		})
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
			// StartInSearchMode defaults to false; setting a Searcher
			// makes the / key toggle search mode but does not auto-enter.
			Searcher: func(input string, idx int) bool {
				return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
			},
			Active:   `> {{ .Name }} ({{ .Id }})`,
			Inactive: `  {{ .Name }} ({{ .Id }})`,
		})
		resCh <- result{idx: idx, err: err}
	}()

	tm.WaitFor("Pick one")
	tm.WaitFor("item-01")
	tm.Golden("01-initial")

	tm.Type("j")
	tm.Type("j")
	tm.Golden("02-after-jj")

	tm.Type("k")
	tm.Golden("03-after-k")

	tm.Type("l")
	tm.Golden("04-after-l-pagedown")

	tm.Type("h")
	tm.Golden("05-after-h-pageup")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
}
