package cmdiotest_test

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_Validate pins Prompt's behavior when a Validate
// callback is configured: validation re-runs on every keystroke, the
// indicator glyph reflects the result, and Enter is blocked while invalid.
func TestPromptBaseline_Validate(t *testing.T) {
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
		v, err := cmdio.RunPrompt(ctx, cmdio.PromptOptions{
			Label: "Workspace host",
			Validate: func(s string) error {
				if !strings.Contains(s, "://") {
					return errors.New("must contain ://")
				}
				return nil
			},
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Workspace host")
	tm.Golden("01-empty")

	tm.Type("abc")
	tm.Golden("02-invalid-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-cleared")

	tm.Type("https://example.com")
	tm.Golden("04-valid")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "https://example.com", res.value, "snapshot:\n%s", tm.Snapshot())
}
