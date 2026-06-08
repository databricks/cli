package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

var (
	defaultActiveTemplate   = `{{.Name | bold}}{{if .IsDefault}} {{ "[default]" | green }}{{end}} ({{.Host|faint}})`
	defaultInactiveTemplate = `{{.Name}}{{if .IsDefault}} [default]{{end}}`
	defaultSelectedTemplate = "{{ \"Using profile\" | faint }}: {{ .Name | bold }}"
)

// SelectConfig configures the interactive profile picker shown by [SelectProfile].
type SelectConfig struct {
	// Label shown above the selection list.
	Label string

	// Profiles to choose from.
	Profiles Profiles

	StartInSearchMode bool

	// Default is the name of the default profile. When non-empty and matching a
	// profile in Profiles, that profile is moved to the top of the list and
	// rendered with IsDefault=true so templates can decorate it (e.g. with a
	// "[default]" tag).
	Default string

	// Go template strings for rendering items. Templates have access to all
	// [Profile] fields, a Cloud method, a PaddedName field that is the profile
	// name right-padded to align with the longest name in the list, and an
	// IsDefault boolean that is true for the entry matching SelectConfig.Default.
	//
	// Defaults:
	//   Active:   `{{.Name | bold}}{{if .IsDefault}} {{ "[default]" | green }}{{end}} ({{.Host|faint}})`
	//   Inactive: `{{.Name}}{{if .IsDefault}} [default]{{end}}`
	//   Selected: `{{ "Using profile" | faint }}: {{ .Name | bold }}`
	ActiveTemplate   string
	InactiveTemplate string
	SelectedTemplate string
}

// selectItem wraps a Profile with display-computed fields available in templates.
type selectItem struct {
	Profile
	PaddedName string
	IsDefault  bool
}

// buildSelectItems returns the list of items to render, with the default profile
// moved to the top and tagged with IsDefault=true. The relative order of the
// other profiles is preserved.
func buildSelectItems(profiles Profiles, defaultName string) []selectItem {
	maxNameLen := 0
	for _, p := range profiles {
		maxNameLen = max(maxNameLen, len(p.Name))
	}

	items := make([]selectItem, 0, len(profiles))
	itemFor := func(p Profile) selectItem {
		return selectItem{
			Profile:    p,
			PaddedName: fmt.Sprintf("%-*s", maxNameLen, p.Name),
			IsDefault:  defaultName != "" && p.Name == defaultName,
		}
	}

	defaultIdx := -1
	if defaultName != "" {
		for i, p := range profiles {
			if p.Name == defaultName {
				defaultIdx = i
				break
			}
		}
	}
	if defaultIdx >= 0 {
		items = append(items, itemFor(profiles[defaultIdx]))
	}
	for i, p := range profiles {
		if i == defaultIdx {
			continue
		}
		items = append(items, itemFor(p))
	}
	return items
}

// SelectProfile shows an interactive profile picker and returns the name of the
// selected profile.
func SelectProfile(ctx context.Context, cfg SelectConfig) (string, error) {
	if len(cfg.Profiles) == 0 {
		return "", errors.New("no profiles configured. Run 'databricks auth login' to create a profile")
	}

	items := buildSelectItems(cfg.Profiles, cfg.Default)

	if cfg.ActiveTemplate == "" {
		cfg.ActiveTemplate = defaultActiveTemplate
	}
	if cfg.InactiveTemplate == "" {
		cfg.InactiveTemplate = defaultInactiveTemplate
	}
	if cfg.SelectedTemplate == "" {
		cfg.SelectedTemplate = defaultSelectedTemplate
	}

	// Build the searcher from the items slice directly so it stays coupled
	// to the Items list passed to cmdio.RunSelect (rather than the original
	// Profiles slice which could diverge if items were ever filtered or reordered).
	searcher := func(input string, index int) bool {
		input = strings.ToLower(input)
		p := items[index].Profile
		return strings.Contains(strings.ToLower(p.Name), input) ||
			strings.Contains(strings.ToLower(p.Host), input) ||
			strings.Contains(strings.ToLower(p.AccountID), input)
	}

	i, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label:             cfg.Label,
		Items:             items,
		StartInSearchMode: cfg.StartInSearchMode,
		Searcher:          searcher,
		LabelTemplate:     "{{ . | faint }}",
		Active:            cfg.ActiveTemplate,
		Inactive:          cfg.InactiveTemplate,
		Selected:          cfg.SelectedTemplate,
	})
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
}
