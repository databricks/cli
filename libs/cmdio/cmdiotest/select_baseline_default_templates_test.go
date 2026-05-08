package cmdiotest_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_DefaultTemplates pins the rendering of
// [cmdio.RunSelect] when no Label / Active / Inactive / Selected
// template is provided — promptui falls back to its built-in defaults,
// which print {{.}} (i.e. Go's default formatting for the item struct
// rather than any specific field).
//
// This mirrors the `databricks selftest tui run-select` plain mode
// (cmd/selftest/tui/select.go: runSelectPlain) and exists so future
// changes to the defaults — or accidental loss of a custom template at
// a call site — produce a visible diff.
func TestSelectBaseline_DefaultTemplates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := context.Background()
	io := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io)

	require.True(t, cmdio.IsPromptSupported(ctx), "prompt support must be detected on the pty")

	// Same data shape as cmd/selftest/tui/fixtures.go buildItems(5).
	items := []cmdio.Tuple{
		{Name: "unity-catalog", Id: "id-01"},
		{Name: "delta-lake", Id: "id-02"},
		{Name: "delta-sharing", Id: "id-03"},
		{Name: "photon", Id: "id-04"},
		{Name: "mlflow", Id: "id-05"},
	}

	type result struct {
		idx int
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		idx, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
			Label: "Pick an item",
			Items: items,
		})
		resCh <- result{idx: idx, err: err}
	}()

	tm.WaitFor("Pick an item")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-after-down")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, 1, res.idx, "snapshot:\n%s", tm.Snapshot())

	tm.Golden("03-after-enter")
}
