package testcli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func (r *Runner) registerFlagCleanup(c *cobra.Command) {
	// Find target command that will be run. Example: if the command run is `databricks fs cp`,
	// target command corresponds to `cp`
	targetCmd, _, err := c.Find(r.args)
	if err != nil && strings.HasPrefix(err.Error(), "unknown command") {
		// even if command is unknown, we can proceed
		require.NotNil(r, targetCmd)
	} else {
		require.NoError(r, err)
	}

	// Force initialization of default flags.
	// These are initialized by cobra at execution time and would otherwise
	// not be cleaned up by the cleanup function below.
	targetCmd.InitDefaultHelpFlag()
	targetCmd.InitDefaultVersionFlag()

	// Restore flag values to their original value on test completion.
	targetCmd.Flags().VisitAll(func(f *pflag.Flag) {
		v := reflect.ValueOf(f.Value)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		// Store copy of the current flag value.
		reset := reflect.New(v.Type()).Elem()
		reset.Set(v)
		r.Cleanup(func() {
			v.Set(reset)
		})
	})
}

// Like [cobraTestRunner.Eventually], but more specific
func (r *Runner) WaitForTextPrinted(text string, timeout time.Duration) {
	r.Eventually(func() bool {
		currentStdout := r.stdout.String()
		return strings.Contains(currentStdout, text)
	}, timeout, 50*time.Millisecond)
}

func (r *Runner) WaitForOutput(text string, timeout time.Duration) {
	require.Eventually(r, func() bool {
		currentStdout := r.stdout.String()
		currentErrout := r.stderr.String()
		return strings.Contains(currentStdout, text) || strings.Contains(currentErrout, text)
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

	// Register cleanup function to restore flags to their original values
	// once test has been executed. This is needed because flag values reside
	// in a global singleton data-structure, and thus subsequent tests might
	// otherwise interfere with each other
	r.registerFlagCleanup(cli)

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
			r.Logf("Error running command: %s", err)
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
				r.Logf("[databricks stdout]: %s", scanner.Text())
			}
		}

		if r.stderr.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(r.stderr.Bytes()))
			for scanner.Scan() {
				r.Logf("[databricks stderr]: %s", scanner.Text())
			}
		}

		// Reset context on command for the next test.
		// These commands are globals so we have to clean up to the best of our ability after each run.
		// See https://github.com/spf13/cobra/blob/a6f198b635c4b18fff81930c40d464904e55b161/command.go#L1062-L1066
		//nolint:staticcheck  // cobra sets the context and doesn't clear it
		cli.SetContext(nil)

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
	r.RunBackground()
	err := <-r.errch
	return r.stdout, r.stderr, err
}

// Like [require.Eventually] but errors if the underlying command has failed.
func (r *Runner) Eventually(condition func() bool, waitFor, tick time.Duration, msgAndArgs ...any) {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	// Kick off condition check immediately.
	go func() { ch <- condition() }()

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
			go func() { ch <- condition() }()
		case v := <-ch:
			if v {
				return
			}
			tick = ticker.C
		}
	}
}

func (r *Runner) RunAndExpectOutput(heredoc string) {
	stdout, _, err := r.Run()
	require.NoError(r, err)
	require.Equal(r, cmdio.Heredoc(heredoc), strings.TrimSpace(stdout.String()))
}

func (r *Runner) RunAndParseJSON(v any) {
	stdout, _, err := r.Run()
	require.NoError(r, err)
	err = json.Unmarshal(stdout.Bytes(), &v)
	require.NoError(r, err)
}

func NewRunner(t testutil.TestingT, args ...string) *Runner {
	return &Runner{
		TestingT: t,

		ctx:  context.Background(),
		args: args,
	}
}

func NewRunnerWithContext(t testutil.TestingT, ctx context.Context, args ...string) *Runner {
	return &Runner{
		TestingT: t,

		ctx:  ctx,
		args: args,
	}
}

func RequireSuccessfulRun(t testutil.TestingT, args ...string) (bytes.Buffer, bytes.Buffer) {
	t.Logf("run args: [%s]", strings.Join(args, ", "))
	r := NewRunner(t, args...)
	stdout, stderr, err := r.Run()
	require.NoError(t, err)
	return stdout, stderr
}

func RequireErrorRun(t testutil.TestingT, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	r := NewRunner(t, args...)
	stdout, stderr, err := r.Run()
	require.Error(t, err)
	return stdout, stderr, err
}
