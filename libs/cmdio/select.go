package cmdio

import (
	"context"

	"github.com/manifoldco/promptui"
)

// SelectOptions configures an interactive single-choice picker shown by
// [RunSelect]. Template strings use text/template syntax and have access
// to the fields of the items in Items.
type SelectOptions struct {
	// Label is shown above the list. Required.
	Label string

	// Items is the slice of values to choose from. Templates reference
	// fields on the element type.
	Items any

	// Searcher, when set, narrows the list as the user types.
	Searcher func(input string, index int) bool

	// StartInSearchMode opens the prompt with the search input focused.
	StartInSearchMode bool

	// LabelTemplate renders Label. Empty uses the default.
	LabelTemplate string

	// Active renders the highlighted item.
	Active string

	// Inactive renders non-highlighted items.
	Inactive string

	// Selected renders the chosen item after the prompt closes.
	Selected string
}

// RunSelect shows an interactive picker and returns the index of the chosen item.
func RunSelect(ctx context.Context, opts SelectOptions) (int, error) {
	c := fromContext(ctx)
	sel := &promptui.Select{
		Label:             opts.Label,
		Items:             opts.Items,
		Searcher:          opts.Searcher,
		StartInSearchMode: opts.StartInSearchMode,
		Templates: &promptui.SelectTemplates{
			Label:    opts.LabelTemplate,
			Active:   opts.Active,
			Inactive: opts.Inactive,
			Selected: opts.Selected,
		},
		Stdin:  c.promptStdin(),
		Stdout: nopWriteCloser{c.err},
	}
	idx, _, err := sel.Run()
	return idx, err
}
