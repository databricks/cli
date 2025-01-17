package cmdio

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"
)

// cmdIO is the private instance, that is not supposed to be accessed
// outside of `cmdio` package. Use the public package-level functions
// to access the inner state.
type cmdIO struct {
	// states if we are in the interactive mode
	// e.g. if stdout is a terminal
	interactive    bool
	outputFormat   flags.Output
	headerTemplate string
	template       string
	in             io.Reader
	out            io.Writer
	err            io.Writer
}

func NewIO(ctx context.Context, outputFormat flags.Output, in io.Reader, out, err io.Writer, headerTemplate, template string) *cmdIO {
	// The check below is similar to color.NoColor but uses the specified err writer.
	dumb := env.Get(ctx, "NO_COLOR") != "" || env.Get(ctx, "TERM") == "dumb"
	if f, ok := err.(*os.File); ok && !dumb {
		dumb = !isatty.IsTerminal(f.Fd()) && !isatty.IsCygwinTerminal(f.Fd())
	}
	return &cmdIO{
		interactive:    !dumb,
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
	return c.interactive
}

// IsTTY detects if io.Writer is a terminal.
func IsTTY(w any) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// IsInTTY detects if the input reader is a terminal.
func IsInTTY(ctx context.Context) bool {
	c := fromContext(ctx)
	return IsTTY(c.in)
}

// IsOutTTY detects if the output writer is a terminal.
func IsOutTTY(ctx context.Context) bool {
	c := fromContext(ctx)
	return IsTTY(c.out)
}

// IsErrTTY detects if the error writer is a terminal.
func IsErrTTY(ctx context.Context) bool {
	c := fromContext(ctx)
	return IsTTY(c.err)
}

// IsTTY detects if stdout is a terminal. It assumes that stderr is terminal as well
func (c *cmdIO) IsTTY() bool {
	f, ok := c.out.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func IsPromptSupported(ctx context.Context) bool {
	// We do not allow prompting in non-interactive mode and in Git Bash on Windows.
	// Likely due to fact that Git Bash does not (correctly support ANSI escape sequences,
	// we cannot use promptui package there.
	// See known issues:
	// - https://github.com/manifoldco/promptui/issues/208
	// - https://github.com/chzyer/readline/issues/191
	// We also do not allow prompting in non-interactive mode,
	// because it's not possible to read from stdin in non-interactive mode.
	return (IsInteractive(ctx) || (IsOutTTY(ctx) && IsInTTY(ctx))) && !IsGitBash(ctx)
}

func IsGitBash(ctx context.Context) bool {
	// Check if the MSYSTEM environment variable is set to "MINGW64"
	msystem := env.Get(ctx, "MSYSTEM")
	if strings.EqualFold(msystem, "MINGW64") {
		// Check for typical Git Bash env variable for prompts
		ps1 := env.Get(ctx, "PS1")
		return strings.Contains(ps1, "MINGW") || strings.Contains(ps1, "MSYSTEM")
	}

	return false
}

type Tuple struct{ Name, Id string }

func (c *cmdIO) Select(items []Tuple, label string) (id string, err error) {
	if !c.interactive {
		return "", fmt.Errorf("expected to have %s", label)
	}

	idx, _, err := (&promptui.Select{
		Label:             label,
		Items:             items,
		HideSelected:      true,
		StartInSearchMode: true,
		Searcher: func(input string, idx int) bool {
			lower := strings.ToLower(items[idx].Name)
			return strings.Contains(lower, input)
		},
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.Id|faint}})`,
			Inactive: `{{.Name}}`,
		},
		Stdin: io.NopCloser(c.in),
	}).Run()
	if err != nil {
		return
	}
	id = items[idx].Id
	return
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
	})

	return prompt.Run()
}

func Secret(ctx context.Context, label string) (value string, err error) {
	c := fromContext(ctx)
	return c.Secret(label)
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
		Stdin:  io.NopCloser(c.in),
		Stdout: nopWriteCloser{c.out},
	}
}

func RunSelect(ctx context.Context, prompt *promptui.Select) (int, string, error) {
	c := fromContext(ctx)
	prompt.Stdin = io.NopCloser(c.in)
	prompt.Stdout = nopWriteCloser{c.err}
	return prompt.Run()
}

func (c *cmdIO) simplePrompt(label string) *promptui.Prompt {
	return &promptui.Prompt{
		Label:  label,
		Stdin:  io.NopCloser(c.in),
		Stdout: nopWriteCloser{c.out},
	}
}

func (c *cmdIO) SimplePrompt(label string) (value string, err error) {
	return c.simplePrompt(label).Run()
}

func SimplePrompt(ctx context.Context, label string) (value string, err error) {
	c := fromContext(ctx)
	return c.SimplePrompt(label)
}

func (c *cmdIO) DefaultPrompt(label, defaultValue string) (value string, err error) {
	prompt := c.simplePrompt(label)
	prompt.Default = defaultValue
	prompt.AllowEdit = true
	return prompt.Run()
}

func DefaultPrompt(ctx context.Context, label, defaultValue string) (value string, err error) {
	c := fromContext(ctx)
	return c.DefaultPrompt(label, defaultValue)
}

func (c *cmdIO) Spinner(ctx context.Context) chan string {
	var sp *spinner.Spinner
	if c.interactive {
		charset := spinner.CharSets[11]
		sp = spinner.New(charset, 200*time.Millisecond,
			spinner.WithWriter(c.err),
			spinner.WithColor("green"))
		sp.Start()
	}
	updates := make(chan string)
	go func() {
		if c.interactive {
			defer sp.Stop()
		}
		for {
			select {
			case <-ctx.Done():
				return
			case x, hasMore := <-updates:
				if c.interactive {
					// `sp`` access is isolated to this method,
					// so it's safe to update it from this goroutine.
					sp.Suffix = " " + x
				}
				if !hasMore {
					return
				}
			}
		}
	}()
	return updates
}

func Spinner(ctx context.Context) chan string {
	c := fromContext(ctx)
	return c.Spinner(ctx)
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
		interactive:  false,
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	})
}
