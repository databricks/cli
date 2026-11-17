package cmdio

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"github.com/databricks/cli/libs/flags"
)

// ErrCtrlC is returned when the user cancels a TUI prompt with Ctrl+C. The
// "^C" string matches the historical wire format; goldens depend on it.
var ErrCtrlC = errors.New("^C")

// runTUI runs a tea.Program through cmdIO's tea program slot so spinners and
// pagers can't fight a prompt for the terminal. Blocks until the model quits.
func (c *cmdIO) runTUI(m tea.Model) (tea.Model, error) {
	opts := []tea.ProgramOption{
		tea.WithInput(c.in),
		tea.WithOutput(c.err),
		tea.WithColorProfile(teaColorProfile(c.capabilities.SupportsStderrColor(), c.err)),
		// Ctrl+C is delivered as a key event so the model can return ErrCtrlC.
		tea.WithoutSignalHandler(),
	}
	// NewTestIO sets a synthetic window size because bubbletea's auto-detect
	// doesn't fire on pipe-backed streams. Nil in production. It must be set
	// up front: Run renders the first frame before any sent message arrives.
	if c.teaWindowSize != nil {
		opts = append(opts, tea.WithWindowSize(c.teaWindowSize.Width, c.teaWindowSize.Height))
	}
	p := NewTeaProgram(m, c.in, opts...)
	c.acquireTeaProgram(p)
	defer c.releaseTeaProgram()
	return p.Run()
}

// cmdIO is the private instance, that is not supposed to be accessed
// outside of `cmdio` package. Use the public package-level functions
// to access the inner state.
//
// Stream Architecture:
//   - in:  stdin for user input (prompts, confirmations)
//   - out: stdout for data output (JSON, tables, command results)
//   - err: stderr for interactive UI (prompts, spinners, logs, diagnostics)
//
// This separation enables piping stdout while maintaining interactivity:
//
//	databricks clusters list --output json | jq  # User sees prompts, jq gets JSON
type cmdIO struct {
	capabilities   Capabilities
	outputFormat   flags.Output
	headerTemplate string
	template       string
	in             io.Reader
	out            io.Writer
	err            io.Writer

	// Bubble Tea program lifecycle management
	teaMu      sync.Mutex
	teaProgram *tea.Program
	teaDone    chan struct{}

	// teaWindowSize, when non-nil, is delivered to the tea.Program before any
	// user input is processed. Populated only by NewTestIO so pipe-backed
	// test runs receive a synthetic WindowSizeMsg.
	teaWindowSize *tea.WindowSizeMsg

	// errWriter wraps err for LogString; built once by stderrWriter.
	errWriterMu sync.Mutex
	errWriter   io.Writer
}

// stderrWriter returns err wrapped to match its color support, built on first
// use. Whether err supports color depends only on this cmdIO's capabilities,
// so the result does not change; the color depth is read from ctx's
// environment on that first call. (sync.OnceValue can't take ctx.)
func (c *cmdIO) stderrWriter(ctx context.Context) io.Writer {
	c.errWriterMu.Lock()
	defer c.errWriterMu.Unlock()
	if c.errWriter == nil {
		r, _ := NewRenderer(ctx, c.err)
		c.errWriter = r.Writer(c.err)
	}
	return c.errWriter
}

func NewIO(ctx context.Context, outputFormat flags.Output, in io.Reader, out, err io.Writer, headerTemplate, template string) *cmdIO {
	return &cmdIO{
		capabilities:   newCapabilities(ctx, in, out, err),
		outputFormat:   outputFormat,
		headerTemplate: headerTemplate,
		template:       template,
		in:             in,
		out:            out,
		err:            err,
	}
}

func IsPromptSupported(ctx context.Context) bool {
	c := fromContext(ctx)
	return c.capabilities.SupportsPrompt()
}

// IsLinePromptSupported reports whether a plain-text line prompt ([Ask]) can
// be shown and answered: stdin, stdout and stderr are terminals, this process
// is in the terminal's foreground process group (a background job that reads
// it is stopped), and no goroutine reads stdin in the background (see
// [MarkStdinReadInBackground]). Unlike IsPromptSupported, it does not depend
// on TERM=dumb or an MSYS2 pty pipe, which only matter for the TUI prompts.
func IsLinePromptSupported(ctx context.Context) bool {
	c := fromContext(ctx)
	if !c.capabilities.SupportsLinePrompt() || stdinReadInBackground.Load() {
		return false
	}
	f, ok := c.in.(*os.File)
	return !ok || inForeground(f)
}

// stdinReadInBackground is set once a goroutine reads stdin for the rest of
// the process.
var stdinReadInBackground atomic.Bool

// MarkStdinReadInBackground records that a goroutine reads os.Stdin and may
// outlive the command, so later line prompts in this process are not asked:
// that goroutine could take their answer.
func MarkStdinReadInBackground() {
	stdinReadInBackground.Store(true)
}

// DiscardInput discards what was typed at the stdin terminal but not read
// yet, so that a prompt asked after a long-running command reads only what is
// typed after the question.
func DiscardInput(ctx context.Context) error {
	c := fromContext(ctx)
	f, ok := c.in.(*os.File)
	if !ok || !c.capabilities.stdinIsTTY {
		return nil
	}
	return discardInput(f)
}

// IsInputTTY reports whether stdin is a terminal, so that a user can type
// input into the command. Unlike IsPromptSupported, it does not depend on
// stderr or TERM=dumb, which only matter for rendering prompts, and it accepts
// MSYS2 pty pipes, which carry line input but not keystrokes.
func IsInputTTY(ctx context.Context) bool {
	c := fromContext(ctx)
	return c.capabilities.stdinIsTTY
}

// IsPagerSupported reports whether stdin, stdout, and stderr are all interactive
// terminals. This is the requirement for a full-screen or navigable output
// program: unlike IsPromptSupported it also checks stdout, so it returns false
// when stdout is piped or redirected.
func IsPagerSupported(ctx context.Context) bool {
	c := fromContext(ctx)
	return c.capabilities.SupportsPager()
}

// HasTerminal reports whether stdin, stdout or stderr is a terminal. Unlike
// GetInteractiveMode it ignores TERM=dumb.
func HasTerminal(ctx context.Context) bool {
	c := fromContext(ctx)
	return c.capabilities.stdinIsTTY || c.capabilities.stdoutIsTTY || c.capabilities.stderrIsTTY
}

// SupportsColor returns true if the given writer supports colored output.
// This checks both TTY status and environment variables (NO_COLOR, TERM=dumb).
func SupportsColor(ctx context.Context, w io.Writer) bool {
	c := fromContext(ctx)
	return c.capabilities.SupportsColor(w)
}

// GetInteractiveMode returns the interactive mode based on terminal capabilities.
// Returns one of: InteractiveModeFull, InteractiveModeOutputOnly, or InteractiveModeNone.
func GetInteractiveMode(ctx context.Context) InteractiveMode {
	c := fromContext(ctx)
	return c.capabilities.InteractiveMode()
}

// NewSpinner creates a new spinner for displaying progress indicators.
// The returned spinner should be closed when done to release resources.
//
// Example:
//
//	sp := cmdio.NewSpinner(ctx)
//	defer sp.Close()
//	for i := range 100 {
//		sp.Update(fmt.Sprintf("processing item %d", i))
//		time.Sleep(100 * time.Millisecond)
//	}
//
// The spinner automatically degrades in non-interactive terminals (no output).
// Context cancellation will automatically close the spinner.
func NewSpinner(ctx context.Context, opts ...SpinnerOption) *spinner {
	c := fromContext(ctx)
	return c.NewSpinner(ctx, opts...)
}

type cmdIOType int

var cmdIOKey cmdIOType

func InContext(ctx context.Context, io *cmdIO) context.Context {
	return context.WithValue(ctx, cmdIOKey, io)
}

// HasIO reports whether a cmdIO is set on this context. Commands can assume it
// is (the root command's PersistentPreRunE installs it), but code that can run
// before or without that hook (e.g. cobra resolves --help and bare invocations
// before running hooks) must check first to avoid the panic in fromContext.
func HasIO(ctx context.Context) bool {
	_, ok := ctx.Value(cmdIOKey).(*cmdIO)
	return ok
}

func fromContext(ctx context.Context) *cmdIO {
	io, ok := ctx.Value(cmdIOKey).(*cmdIO)
	if !ok {
		panic("no cmdIO found in the context. Please report it as an issue")
	}
	return io
}

// Mocks the context with a cmdio object that discards all output.
func MockDiscard(ctx context.Context) context.Context {
	return InContext(ctx, &cmdIO{
		capabilities: Capabilities{
			stdinIsTTY:  false,
			stdoutIsTTY: false,
			stderrIsTTY: false,
		},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	})
}

// acquireTeaProgram waits for any existing tea.Program to finish, then registers the new one.
// This ensures only one tea.Program runs at a time (e.g., sequential spinners).
func (c *cmdIO) acquireTeaProgram(p *tea.Program) {
	c.teaMu.Lock()
	defer c.teaMu.Unlock()

	// Wait for existing program to finish
	// Receive with teaMu released: releaseTeaProgram locks teaMu to close
	// teaDone, so waiting while holding it would deadlock. Loop because another
	// acquirer may register a new program before the lock is reacquired.
	for c.teaDone != nil {
		done := c.teaDone
		c.teaMu.Unlock()
		<-done
		c.teaMu.Lock()
	}

	// Register new program
	c.teaProgram = p
	c.teaDone = make(chan struct{})
}

// releaseTeaProgram signals that the current tea.Program has finished.
func (c *cmdIO) releaseTeaProgram() {
	c.teaMu.Lock()
	defer c.teaMu.Unlock()

	if c.teaDone != nil {
		close(c.teaDone)
		c.teaDone = nil
	}
	c.teaProgram = nil
}

// Wait blocks until any active tea.Program finishes.
// This should be called before command termination to ensure terminal state is restored.
func Wait(ctx context.Context) {
	c := fromContext(ctx)
	c.teaMu.Lock()
	done := c.teaDone
	c.teaMu.Unlock()

	if done != nil {
		<-done
	}
}
