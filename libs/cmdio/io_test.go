package cmdio

import (
	"bytes"
	"io"
	"strings"
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
	// Prompts are not supported without stderr as a terminal,
	// but a user can still type into stdin.
	ctx := InContext(t.Context(), &cmdIO{capabilities: Capabilities{stdinIsTTY: true}})
	assert.True(t, IsInputTTY(ctx))
	assert.False(t, IsPromptSupported(ctx))

	ctx = InContext(t.Context(), &cmdIO{capabilities: Capabilities{stderrIsTTY: true}})
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

func TestIsLinePromptSupported(t *testing.T) {
	ctx := t.Context()
	terminal := InContext(ctx, NewIO(ctx, flags.OutputText, FakeTTYReader(strings.NewReader("")), FakeTTY(io.Discard), FakeTTY(io.Discard), "", ""))
	assert.True(t, IsLinePromptSupported(terminal))

	piped := InContext(ctx, NewIO(ctx, flags.OutputText, strings.NewReader(""), FakeTTY(io.Discard), FakeTTY(io.Discard), "", ""))
	assert.False(t, IsLinePromptSupported(piped))

	t.Cleanup(func() { stdinReadInBackground.Store(false) })
	MarkStdinReadInBackground()
	assert.False(t, IsLinePromptSupported(terminal), "another goroutine reads stdin")
}

// CI and CLICOLOR=0 make termenv's profile Ascii on a terminal, but cmdio's
// color helpers still color LogString output there, as they did on v1.
func TestLogStringKeepsHelperColorsWhenProfileIsAscii(t *testing.T) {
	for _, kv := range [][2]string{{"CI", "1"}, {"CLICOLOR", "0"}} {
		t.Run(kv[0], func(t *testing.T) {
			t.Setenv(kv[0], kv[1])
			var stderr bytes.Buffer
			c := NewTestIO(nil, io.Discard, FakeTTY(&stderr))
			ctx := InContext(t.Context(), c)

			LogString(ctx, "\x1b[33mwarning\x1b[0m")

			// The writer re-encodes SGR, so the reset comes out in its short form.
			assert.Equal(t, "\x1b[33mwarning\x1b[m\n", stderr.String())
		})
	}
}
