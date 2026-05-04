package cmdio

import (
	"context"

	"github.com/manifoldco/promptui"
)

// PromptOptions configures a single-line text prompt shown by [RunPrompt].
type PromptOptions struct {
	// Label is shown before the input field. Required.
	Label string

	// Default is the value pre-filled in the input field.
	Default string

	// Mask, when non-zero, replaces typed characters with the given rune
	// (use '*' for password-style input).
	Mask rune

	// AllowEdit lets the user edit Default rather than overwriting it.
	AllowEdit bool

	// Validate, when set, is called on every keystroke; returning a non-nil
	// error keeps the prompt open and shows the error to the user.
	Validate func(input string) error
}

// RunPrompt shows a single-line text prompt and returns the entered value.
func RunPrompt(ctx context.Context, opts PromptOptions) (string, error) {
	c := fromContext(ctx)
	p := promptui.Prompt{
		Label:     opts.Label,
		Default:   opts.Default,
		Mask:      opts.Mask,
		AllowEdit: opts.AllowEdit,
		Validate:  opts.Validate,
		Stdin:     c.promptStdin(),
		Stdout:    nopWriteCloser{c.err},
	}
	return p.Run()
}
