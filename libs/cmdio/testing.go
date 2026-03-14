package cmdio

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/databricks/cli/libs/flags"
)

type Test struct {
	Done context.CancelFunc

	Stdin  *bufio.Writer
	Stdout *bufio.Reader
	Stderr *bufio.Reader
}

type TestOptions struct {
	// PromptSupported indicates whether IsPromptSupported should return true
	// for the test context. If false (default), prompting will fail as it would
	// in a non-interactive environment (e.g., CI).
	PromptSupported bool
}

// SetupTest creates a cmdio context with pipes for stdin/stdout/stderr.
// This is useful for testing interactive I/O operations.
//
// By default, IsPromptSupported returns false for the test context because
// pipes are not TTYs. To test prompt logic, pass TestOptions{PromptSupported: true}.
func SetupTest(ctx context.Context, opts TestOptions) (context.Context, *Test) {
	rin, win := io.Pipe()
	rout, wout := io.Pipe()
	rerr, werr := io.Pipe()

	cmdio := &cmdIO{
		capabilities: Capabilities{
			stdinIsTTY:  opts.PromptSupported,
			stdoutIsTTY: opts.PromptSupported,
			stderrIsTTY: true,
			color:       opts.PromptSupported,
			isGitBash:   false,
		},
		in:  rin,
		out: wout,
		err: werr,
	}

	ctx, cancel := context.WithCancel(ctx)
	ctx = InContext(ctx, cmdio)

	// Wait for context to be done, so we can drain stdin and close the pipes.
	go func() {
		<-ctx.Done()
		rin.Close()
		wout.Close()
		werr.Close()
	}()

	return ctx, &Test{
		Done:   cancel,
		Stdin:  bufio.NewWriter(win),
		Stdout: bufio.NewReader(rout),
		Stderr: bufio.NewReader(rerr),
	}
}

// NewTestContextWithStdout creates a cmdio context that captures stdout output.
// Stderr is discarded. Use this for testing data output and results.
func NewTestContextWithStdout(ctx context.Context) (context.Context, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	cmdIO := NewIO(ctx, flags.OutputText, nil, stdout, io.Discard, "", "")
	return InContext(ctx, cmdIO), stdout
}

// NewTestContextWithStderr creates a cmdio context that captures stderr output.
// Stdout is discarded. Use this for testing diagnostics, logs, and error messages.
func NewTestContextWithStderr(ctx context.Context) (context.Context, *bytes.Buffer) {
	stderr := &bytes.Buffer{}
	cmdIO := NewIO(ctx, flags.OutputText, nil, io.Discard, stderr, "", "")
	return InContext(ctx, cmdIO), stderr
}
