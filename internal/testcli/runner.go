package testcli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
)

// Helper for running the root command in the background.
// It ensures that the background goroutine terminates upon
// test completion through cancelling the command context.
type Runner struct {
	testutil.TestingT

	args   []string
	stdout bytes.Buffer
	stderr bytes.Buffer
	stdinR *io.PipeReader
	stdinW *io.PipeWriter

	ctx context.Context

	// Line-by-line output.
	// Background goroutines populate these channels by reading from stdout/stderr pipes.
	StdoutLines <-chan string
	StderrLines <-chan string

	errch <-chan error

	Verbose bool
}

func consumeLines(ctx context.Context, wg *sync.WaitGroup, r io.Reader) <-chan string {
	ch := make(chan string, 30000)
	wg.Add(1)
	go func() {
		defer close(ch)
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			// We expect to be able to always send these lines into the channel.
			// If we can't, it means the channel is full and likely there is a problem
			// in either the test or the code under test.
			select {
			case <-ctx.Done():
				return
			case ch <- scanner.Text():
				continue
			default:
				panic("line buffer is full")
			}
		}
	}()
	return ch
}

// Like [Runner.Eventually], but more specific
func (r *Runner) WaitForTextPrinted(text string, timeout time.Duration) {
	r.Eventually(func() bool {
		currentStdout := r.stdout.String()
		return strings.Contains(currentStdout, text)
	}, timeout, 50*time.Millisecond)
}

func (r *Runner) WithStdin() {
	reader, writer := io.Pipe()
	r.stdinR = reader
	r.stdinW = writer
}

func (r *Runner) CloseStdin() {
	if r.stdinW == nil {
		panic("no standard input configured")
	}
	r.stdinW.Close()
}

func (r *Runner) SendText(text string) {
	if r.stdinW == nil {
		panic("no standard input configured")
	}
	_, err := r.stdinW.Write([]byte(text + "\n"))
	if err != nil {
		panic("Failed to to write to t.stdinW")
	}
}

func (r *Runner) RunBackground() {
	var stdoutR, stderrR io.Reader
	var stdoutW, stderrW io.WriteCloser
	stdoutR, stdoutW = io.Pipe()
	stderrR, stderrW = io.Pipe()
	ctx := cmdio.NewContext(r.ctx, &cmdio.Logger{
		Mode:   flags.ModeAppend,
		Reader: bufio.Reader{},
		Writer: stderrW,
	})

	cli := cmd.New(ctx)
	cli.SetOut(stdoutW)
	cli.SetErr(stderrW)
	cli.SetArgs(r.args)
	if r.stdinW != nil {
		cli.SetIn(r.stdinR)
	}

	errch := make(chan error)
	ctx, cancel := context.WithCancel(ctx)

	// Tee stdout/stderr to buffers.
	stdoutR = io.TeeReader(stdoutR, &r.stdout)
	stderrR = io.TeeReader(stderrR, &r.stderr)

	// Consume stdout/stderr line-by-line.
	var wg sync.WaitGroup
	r.StdoutLines = consumeLines(ctx, &wg, stdoutR)
	r.StderrLines = consumeLines(ctx, &wg, stderrR)

	// Run command in background.
	go func() {
		err := root.Execute(ctx, cli)
		if err != nil {
			if r.Verbose {
				r.Logf("Error running command: %s", err)
			}
		}

		// Close pipes to signal EOF.
		stdoutW.Close()
		stderrW.Close()

		// Wait for the [consumeLines] routines to finish now that
		// the pipes they're reading from have closed.
		wg.Wait()

		if r.stdout.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(r.stdout.Bytes()))
			for scanner.Scan() {
				if r.Verbose {
					r.Logf("[databricks stdout]: %s", scanner.Text())
				}
			}
		}

		if r.stderr.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(r.stderr.Bytes()))
			for scanner.Scan() {
				if r.Verbose {
					r.Logf("[databricks stderr]: %s", scanner.Text())
				}
			}
		}

		// Make caller aware of error.
		errch <- err
		close(errch)
	}()

	// Ensure command terminates upon test completion (success or failure).
	r.Cleanup(func() {
		// Signal termination of command.
		cancel()
		// Wait for goroutine to finish.
		<-errch
	})

	r.errch = errch
}

func (r *Runner) Run() (bytes.Buffer, bytes.Buffer, error) {
	r.Helper()
	var stdout, stderr bytes.Buffer
	ctx := cmdio.NewContext(r.ctx, &cmdio.Logger{
		Mode:   flags.ModeAppend,
		Reader: bufio.Reader{},
		Writer: &stderr,
	})

	cli := cmd.New(ctx)
	cli.SetOut(&stdout)
	cli.SetErr(&stderr)
	cli.SetArgs(r.args)

	if r.Verbose {
		r.Logf("  args: %s", strings.Join(r.args, ", "))
	}

	err := root.Execute(ctx, cli)
	if err != nil {
		if r.Verbose {
			r.Logf(" error: %s", err)
		}
	}

	if stdout.Len() > 0 {
		// Make a copy of the buffer such that it remains "unread".
		scanner := bufio.NewScanner(bytes.NewBuffer(stdout.Bytes()))
		for scanner.Scan() {
			if r.Verbose {
				r.Logf("stdout: %s", scanner.Text())
			}
		}
	}

	if stderr.Len() > 0 {
		// Make a copy of the buffer such that it remains "unread".
		scanner := bufio.NewScanner(bytes.NewBuffer(stderr.Bytes()))
		for scanner.Scan() {
			if r.Verbose {
				r.Logf("stderr: %s", scanner.Text())
			}
		}
	}

	return stdout, stderr, err
}

// Like [require.Eventually] but errors if the underlying command has failed.
func (r *Runner) Eventually(condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) {
	r.Helper()
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	// Ensure all the goroutines created by this function are cleaned up.
	// If we do not have this check it is possible that multiple goroutines are created,
	// one of them returns and the test terminates. In that scenario if any of the other
	// goroutines use the *testing.T interface, the resulting panic will bring down the
	// entire test runner.
	var wg sync.WaitGroup
	defer wg.Wait()

	// Kick off condition check immediately.
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- condition()
	}()

	for tick := ticker.C; ; {
		select {
		case err := <-r.errch:
			require.Fail(r, "Command failed", err)
			return
		case <-timer.C:
			require.Fail(r, "Condition never satisfied", msgAndArgs...)
			return
		case <-tick:
			tick = nil
			wg.Add(1)
			go func() {
				defer wg.Done()
				ch <- condition()
			}()
		case v := <-ch:
			if v {
				return
			}
			tick = ticker.C
		}
	}
}

func (r *Runner) RunAndExpectOutput(heredoc string) {
	r.Helper()
	stdout, _, err := r.Run()
	require.NoError(r, err)
	require.Equal(r, cmdio.Heredoc(heredoc), strings.TrimSpace(stdout.String()))
}

func (r *Runner) RunAndParseJSON(v any) {
	r.Helper()
	stdout, _, err := r.Run()
	require.NoError(r, err)
	err = json.Unmarshal(stdout.Bytes(), &v)
	require.NoError(r, err)
}

func NewRunner(t testutil.TestingT, ctx context.Context, args ...string) *Runner {
	return &Runner{
		TestingT: t,

		ctx:     ctx,
		args:    args,
		Verbose: true,
	}
}

func RequireSuccessfulRun(t testutil.TestingT, ctx context.Context, args ...string) (bytes.Buffer, bytes.Buffer) {
	t.Helper()
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	require.NoError(t, err)
	return stdout, stderr
}

func RequireErrorRun(t testutil.TestingT, ctx context.Context, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	t.Helper()
	r := NewRunner(t, ctx, args...)
	stdout, stderr, err := r.Run()
	require.Error(t, err)
	return stdout, stderr, err
}
