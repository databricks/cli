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
	interactive  bool
	outputFormat flags.Output
	template     string
	in           io.Reader
	out          io.Writer
	err          io.Writer
}

func NewIO(outputFormat flags.Output, in io.Reader, out io.Writer, err io.Writer, template string) *cmdIO {
	// The check below is similar to color.NoColor but uses the specified err writer.
	dumb := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	if f, ok := err.(*os.File); ok && !dumb {
		dumb = !isatty.IsTerminal(f.Fd()) && !isatty.IsCygwinTerminal(f.Fd())
	}
	return &cmdIO{
		interactive:  !dumb,
		outputFormat: outputFormat,
		template:     template,
		in:           in,
		out:          out,
		err:          err,
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

func Render(ctx context.Context, v any) error {
	c := fromContext(ctx)
	return RenderWithTemplate(ctx, v, c.template)
}

func RenderWithTemplate(ctx context.Context, v any, template string) error {
	// TODO: add terminal width & white/dark theme detection
	c := fromContext(ctx)
	switch c.outputFormat {
	case flags.OutputJSON:
		return renderJson(c.out, v)
	case flags.OutputText:
		if template != "" {
			return renderTemplate(c.out, template, v)
		}
		return renderJson(c.out, v)
	default:
		return fmt.Errorf("invalid output format: %s", c.outputFormat)
	}
}

func RenderJson(ctx context.Context, v any) error {
	c := fromContext(ctx)
	if c.outputFormat == flags.OutputJSON {
		return renderJson(c.out, v)
	}
	return nil
}

func RenderReader(ctx context.Context, r io.Reader) error {
	c := fromContext(ctx)
	switch c.outputFormat {
	case flags.OutputJSON:
		return fmt.Errorf("json output not supported")
	case flags.OutputText:
		_, err := io.Copy(c.out, r)
		return err
	default:
		return fmt.Errorf("invalid output format: %s", c.outputFormat)
	}
}

type tuple struct{ Name, Id string }

func (c *cmdIO) Select(names map[string]string, label string) (id string, err error) {
	if !c.interactive {
		return "", fmt.Errorf("expected to have %s", label)
	}
	var items []tuple
	for k, v := range names {
		items = append(items, tuple{k, v})
	}
	slices.SortFunc(items, func(a, b tuple) int {
		return strings.Compare(a.Name, b.Name)
	})
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

func Select[V any](ctx context.Context, names map[string]V, label string) (id string, err error) {
	c := fromContext(ctx)
	stringNames := map[string]string{}
	for k, v := range names {
		stringNames[k] = fmt.Sprint(v)
	}
	return c.Select(stringNames, label)
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
