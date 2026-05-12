package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSecretBaseline_EmptyEnter pins Secret's behavior when the user
// presses Enter immediately without typing anything.
func TestSecretBaseline_EmptyEnter(t *testing.T) {
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

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	t.Logf("value: %q", res.value)
	t.Logf("error: %v", res.err)
}
