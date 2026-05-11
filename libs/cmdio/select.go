package cmdio

import (
	"context"
	"fmt"
	"slices"
	"strings"

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

	// HideHelp hides the navigation help line shown by promptui by default.
	HideHelp bool

	// HideSelected hides the rendered selection after the prompt closes.
	HideSelected bool

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
		HideHelp:          opts.HideHelp,
		HideSelected:      opts.HideSelected,
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

type Tuple struct{ Name, Id string }

// Select shows a selection prompt where the user can pick one of the name/id
// items. The items are sorted alphabetically by name.
func Select[V any](ctx context.Context, names map[string]V, label string) (string, error) {
	items := make([]Tuple, 0, len(names))
	for k, v := range names {
		items = append(items, Tuple{k, fmt.Sprint(v)})
	}
	slices.SortFunc(items, func(a, b Tuple) int {
		return strings.Compare(a.Name, b.Name)
	})
	return SelectOrdered(ctx, items, label)
}

// SelectOrdered shows a selection prompt where the user can pick one of the
// name/id items. The items appear in the order specified in the "items"
// argument.
func SelectOrdered(ctx context.Context, items []Tuple, label string) (string, error) {
	c := fromContext(ctx)
	if !c.capabilities.SupportsInteractive() {
		return "", fmt.Errorf("expected to have %s", label)
	}
	idx, err := RunSelect(ctx, SelectOptions{
		Label:             label,
		Items:             items,
		HideSelected:      true,
		StartInSearchMode: true,
		Searcher: func(input string, idx int) bool {
			return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
		},
		Active:   `{{.Name | bold}} ({{.Id|faint}})`,
		Inactive: `{{.Name}}`,
	})
	if err != nil {
		return "", err
	}
	return items[idx].Id, nil
}
