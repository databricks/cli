package cmdio

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/flags"
)

// errCtrlC is returned when the user cancels a TUI prompt with Ctrl+C. The
// "^C" string matches the historical wire format; goldens depend on it.
var errCtrlC = errors.New("^C")

// runTUI runs a tea.Program through cmdIO's tea program slot so spinners and
// pagers can't fight a prompt for the terminal. Blocks until the model quits.
func (c *cmdIO) runTUI(m tea.Model) (tea.Model, error) {
	opts := []tea.ProgramOption{
		tea.WithInput(c.in),
		tea.WithOutput(c.err),
		// Ctrl+C is delivered as a key event so the model can return errCtrlC.
		tea.WithoutSignalHandler(),
	}
	if c.teaFPS > 0 {
		opts = append(opts, tea.WithFPS(c.teaFPS))
	}
	p := tea.NewProgram(m, opts...)
	c.acquireTeaProgram(p)
	defer c.releaseTeaProgram()
	// Drain any pre-queued messages set by NewTestIO. Empty for production.
	for _, msg := range c.teaInitialMsgs {
		go p.Send(msg)
	}
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

	// teaInitialMsgs is delivered to the tea.Program before any user input
	// is processed. Populated only by NewTestIO so pipe-backed test runs
	// receive a synthetic WindowSizeMsg.
	teaInitialMsgs []tea.Msg

	// teaFPS, when non-zero, is passed to tea.WithFPS. Tests crank this up
	// so the renderer doesn't sit on a 60 FPS tick between keystrokes.
	teaFPS int
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
			color:       false,
			isGitBash:   false,
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
	if c.teaDone != nil {
		<-c.teaDone
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
