package cmdio

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/flags"
	"github.com/manifoldco/promptui"
)

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

func IsInteractive(ctx context.Context) bool {
	c := fromContext(ctx)
	return c.capabilities.SupportsInteractive()
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

type Tuple struct{ Name, Id string }

func (c *cmdIO) Select(items []Tuple, label string) (id string, err error) {
	if !c.capabilities.SupportsInteractive() {
		return "", fmt.Errorf("expected to have %s", label)
	}

	idx, _, err := (&promptui.Select{
		Label:             label,
		Items:             items,
		HideSelected:      true,
		StartInSearchMode: true,
		Searcher: func(input string, idx int) bool {
			lower := strings.ToLower(items[idx].Name)
			return strings.Contains(lower, strings.ToLower(input))
		},
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.Id|faint}})`,
			Inactive: `{{.Name}}`,
		},
		Stdin:  c.promptStdin(),
		Stdout: nopWriteCloser{c.err},
	}).Run()
	if err != nil {
		return id, err
	}
	id = items[idx].Id
	return id, err
}

// Show a selection prompt where the user can pick one of the name/id items.
// The items are sorted alphabetically by name.
func Select[V any](ctx context.Context, names map[string]V, label string) (id string, err error) {
	c := fromContext(ctx)
	var items []Tuple
	for k, v := range names {
		items = append(items, Tuple{k, fmt.Sprint(v)})
	}
	slices.SortFunc(items, func(a, b Tuple) int {
		return strings.Compare(a.Name, b.Name)
	})
	return c.Select(items, label)
}

// Show a selection prompt where the user can pick one of the name/id items.
// The items appear in the order specified in the "items" argument.
func SelectOrdered(ctx context.Context, items []Tuple, label string) (id string, err error) {
	c := fromContext(ctx)
	return c.Select(items, label)
}

func (c *cmdIO) Secret(label string) (value string, err error) {
	prompt := (promptui.Prompt{
		Label:       label,
		Mask:        '*',
		HideEntered: true,
		Stdin:       c.promptStdin(),
		Stdout:      nopWriteCloser{c.err},
	})

	return prompt.Run()
}

func Secret(ctx context.Context, label string) (value string, err error) {
	c := fromContext(ctx)
	return c.Secret(label)
}

// promptStdin returns the stdin reader for use with promptui.
// If the reader is os.Stdin, it returns nil to let the underlying readline
// library use its platform-specific default. On Windows, this is critical
// because readline's default uses ReadConsoleInputW to read arrow keys
// as virtual key events. Passing a wrapped os.Stdin would bypass this
// and break arrow key navigation in selection prompts.
func (c *cmdIO) promptStdin() io.ReadCloser {
	if c.in == os.Stdin {
		return nil
	}
	return io.NopCloser(c.in)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func Prompt(ctx context.Context) *promptui.Prompt {
	c := fromContext(ctx)
	return &promptui.Prompt{
		Stdin:  c.promptStdin(),
		Stdout: nopWriteCloser{c.err},
	}
}

func RunSelect(ctx context.Context, prompt *promptui.Select) (int, string, error) {
	c := fromContext(ctx)
	prompt.Stdin = c.promptStdin()
	prompt.Stdout = nopWriteCloser{c.err}
	return prompt.Run()
}

func Spinner(ctx context.Context) chan string {
	c := fromContext(ctx)
	return c.Spinner(ctx)
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
func NewSpinner(ctx context.Context) *spinner {
	c := fromContext(ctx)
	return c.NewSpinner(ctx)
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
