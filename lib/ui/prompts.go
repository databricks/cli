package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/exp/slices"
)

type tuple struct{ Name, Id string }

func PromptValue[V any](stdin io.Reader, names map[string]V, label string) (id string, err error) {
	if !Interactive {
		return "", fmt.Errorf("expected to have %s", label)
	}
	var items []tuple
	for k, v := range names {
		items = append(items, tuple{k, fmt.Sprint(v)})
	}
	slices.SortFunc(items, func(a, b tuple) bool {
		return a.Name < b.Name
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
		Stdin: io.NopCloser(stdin),
	}).Run()
	if err != nil {
		return
	}
	id = items[idx].Id
	return
}
