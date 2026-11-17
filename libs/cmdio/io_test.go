package cmdio

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestAcquireTeaProgramWaitsForRelease(t *testing.T) {
	c := &cmdIO{}
	// acquireTeaProgram only stores the pointer, so a nil program suffices.
	c.acquireTeaProgram(nil)

	acquired := make(chan struct{})
	go func() {
		c.acquireTeaProgram(nil)
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second acquireTeaProgram returned while the first program was still active")
	case <-time.After(50 * time.Millisecond):
	}

	// Release on a goroutine so a regressed deadlock fails the timeout below
	// instead of hanging the test.
	go c.releaseTeaProgram()

	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("second acquireTeaProgram did not return after the first program was released")
	}

	c.releaseTeaProgram()
}

func TestIsInputTTY(t *testing.T) {
	// Prompts are not supported without stderr as a terminal and with color,
	// but a user can still type into stdin.
	ctx := InContext(t.Context(), &cmdIO{capabilities: Capabilities{stdinIsTTY: true}})
	assert.True(t, IsInputTTY(ctx))
	assert.False(t, IsPromptSupported(ctx))

	ctx = InContext(t.Context(), &cmdIO{capabilities: Capabilities{stderrIsTTY: true, color: true}})
	assert.False(t, IsInputTTY(ctx))
}

func TestHasTerminal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      io.Reader
		out     io.Writer
		err     io.Writer
		noColor bool
		want    bool
	}{
		{name: "no terminal", out: io.Discard, err: io.Discard},
		{name: "stdin", in: FakeTTYReader(&bytes.Buffer{}), out: io.Discard, err: io.Discard, want: true},
		{name: "stdout", out: FakeTTY(io.Discard), err: io.Discard, want: true},
		{name: "stderr", out: io.Discard, err: FakeTTY(io.Discard), want: true},
		{name: "stderr with NO_COLOR", out: io.Discard, err: FakeTTY(io.Discard), noColor: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.noColor {
				ctx = env.Set(ctx, "NO_COLOR", "1")
			}
			ctx = InContext(ctx, NewIO(ctx, flags.OutputText, tc.in, tc.out, tc.err, "", ""))
			assert.Equal(t, tc.want, HasTerminal(ctx))
		})
	}
}
