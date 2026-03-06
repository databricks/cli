package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/manifoldco/promptui"
)

var (
	defaultActiveTemplate   = `{{.Name | bold}} ({{.Host|faint}})`
	defaultInactiveTemplate = `{{.Name}}`
	defaultSelectedTemplate = "{{ \"Using profile\" | faint }}: {{ .Name | bold }}"
)

// SelectConfig configures the interactive profile picker shown by [SelectProfile].
type SelectConfig struct {
	// Label shown above the selection list.
	Label string

	// Profiles to choose from.
	Profiles Profiles

	StartInSearchMode bool

	// Go template strings for rendering items. Templates have access to all
	// [Profile] fields, a Cloud method, and a PaddedName field that is the
	// profile name right-padded to align with the longest name in the list.
	//
	// Defaults:
	//   Active:   `{{.Name | bold}} ({{.Host|faint}})`
	//   Inactive: `{{.Name}}`
	//   Selected: `{{ "Using profile" | faint }}: {{ .Name | bold }}`
	ActiveTemplate   string
	InactiveTemplate string
	SelectedTemplate string
}

// selectItem wraps a Profile with display-computed fields available in templates.
type selectItem struct {
	Profile
	PaddedName string
}

// SelectProfile shows an interactive profile picker and returns the name of the
// selected profile.
func SelectProfile(ctx context.Context, cfg SelectConfig) (string, error) {
	if len(cfg.Profiles) == 0 {
		return "", errors.New("no profiles configured. Run 'databricks auth login' to create a profile")
	}

	maxNameLen := 0
	for _, p := range cfg.Profiles {
		if len(p.Name) > maxNameLen {
			maxNameLen = len(p.Name)
		}
	}

	items := make([]selectItem, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		items[i] = selectItem{
			Profile:    p,
			PaddedName: fmt.Sprintf("%-*s", maxNameLen, p.Name),
		}
	}

	if cfg.ActiveTemplate == "" {
		cfg.ActiveTemplate = defaultActiveTemplate
	}
	if cfg.InactiveTemplate == "" {
		cfg.InactiveTemplate = defaultInactiveTemplate
	}
	if cfg.SelectedTemplate == "" {
		cfg.SelectedTemplate = defaultSelectedTemplate
	}

	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             cfg.Label,
		Items:             items,
		StartInSearchMode: cfg.StartInSearchMode,
		Searcher:          cfg.Profiles.SearchCaseInsensitive,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   cfg.ActiveTemplate,
			Inactive: cfg.InactiveTemplate,
			Selected: cfg.SelectedTemplate,
		},
	})
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
}
