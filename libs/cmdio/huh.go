package cmdio

import (
	"context"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// RunHuhField runs one huh field the way field.Run does, but with the program
// options cmdio's prompts use: the context's input and stderr, and the color
// profile of stderr, so a form matches the other TUIs. huh's WithProgramOptions
// replaces its own defaults, which is why the input and output are passed too.
//
// The reply guard is not installed. It has to inject a message when the
// terminal's answer times out, and a huh program's Send is not reachable from
// a program filter, so a missed answer would hang the quit.
func RunHuhField(ctx context.Context, field huh.Field) error {
	return RunHuhForm(ctx, huh.NewForm(huh.NewGroup(field)).WithShowHelp(false))
}

// RunHuhForm runs form with the program options described on [RunHuhField].
func RunHuhForm(ctx context.Context, form *huh.Form) error {
	return form.WithProgramOptions(huhProgramOptions(ctx)...).Run()
}

func huhProgramOptions(ctx context.Context) []tea.ProgramOption {
	c, _ := ctx.Value(cmdIOKey).(*cmdIO)
	var in io.Reader = os.Stdin
	var errW io.Writer = os.Stderr
	if c != nil {
		in, errW = c.in, c.err
	}
	return []tea.ProgramOption{
		tea.WithInput(in),
		tea.WithOutput(errW),
		tea.WithColorProfile(teaColorProfile(SupportsColor(ctx, errW), errW)),
	}
}
