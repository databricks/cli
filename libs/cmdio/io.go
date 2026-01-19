package cmdio

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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
		Stdin:  io.NopCloser(c.in),
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
		Stdin:       io.NopCloser(c.in),
		Stdout:      nopWriteCloser{c.err},
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
		Stdout: nopWriteCloser{c.err},
	}
}

func RunSelect(ctx context.Context, prompt *promptui.Select) (int, string, error) {
	c := fromContext(ctx)
	prompt.Stdin = io.NopCloser(c.in)
	prompt.Stdout = nopWriteCloser{c.err}
	return prompt.Run()
}

func (c *cmdIO) Spinner(ctx context.Context) chan string {
	var sp *spinner.Spinner
	if c.capabilities.SupportsInteractive() {
		charset := spinner.CharSets[11]
		sp = spinner.New(charset, 200*time.Millisecond,
			spinner.WithWriter(c.err),
			spinner.WithColor("green"))
		sp.Start()
	}
	updates := make(chan string)
	go func() {
		if c.capabilities.SupportsInteractive() {
			defer sp.Stop()
		}
		for {
			select {
			case <-ctx.Done():
				return
			case x, hasMore := <-updates:
				if c.capabilities.SupportsInteractive() {
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
