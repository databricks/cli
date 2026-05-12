package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretBaseline_CtrlC pins Secret's behavior when the user cancels
// with Ctrl+C after typing a few characters.
func TestSecretBaseline_CtrlC(t *testing.T) {
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

	type result struct {
		value string
		err   error
	}
	resCh := make(chan result, 1)
	go func() {
		v, err := cmdio.Secret(ctx, "Personal access token")
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type("abc")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyCtrlC)

	res := <-resCh
	require.Error(t, res.err)
	t.Logf("error: %v", res.err)
	t.Logf("value: %q", res.value)
	assert.Empty(t, res.value, "snapshot:\n%s", tm.Snapshot())
}
