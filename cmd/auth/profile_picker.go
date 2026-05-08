package auth

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// profilePickerResult represents the user's choice from pickAuthProfile.
type profilePickerResult int

const (
	profilePickerProfile   profilePickerResult = iota // an existing profile was picked
	profilePickerCreateNew                            // user chose "Create a new profile"
	profilePickerEnterHost                            // user chose "Enter a host URL manually"
)

const (
	profilePickerCreateNewLabel = "Create a new profile"
	profilePickerEnterHostLabel = "Enter a host URL manually"
)

// profilePickerOptions configures pickAuthProfile.
type profilePickerOptions struct {
	// Label shown above the picker.
	Label string

	// Default is the name of the default profile. When set, it is moved to the
	// top of the list and decorated with "[default]".
	Default string

	// IncludeExtras appends "Create a new profile" and "Enter a host URL
	// manually" entries after the profile list.
	IncludeExtras bool
}

// pickerItem is a single entry rendered by the picker. It can be either a real
// profile or one of the extra action entries (Create new / Enter host).
type pickerItem struct {
	Name      string
	Host      string
	AccountID string
	IsDefault bool

	// IsExtra distinguishes action entries (Create new / Enter host) from
	// real profiles, so a profile that happens to share a label name still
	// resolves correctly.
	IsExtra bool
	Extra   profilePickerResult
}

// buildPickerItems returns the items shown by pickAuthProfile, with the default
// profile moved to the top and the extras appended (when requested).
func buildPickerItems(profiles profile.Profiles, defaultName string, includeExtras bool) []pickerItem {
	defaultIdx := -1
	if defaultName != "" {
		for i, p := range profiles {
			if p.Name == defaultName {
				defaultIdx = i
				break
			}
		}
	}

	itemFor := func(p profile.Profile, isDefault bool) pickerItem {
		return pickerItem{
			Name:      p.Name,
			Host:      p.Host,
			AccountID: p.AccountID,
			IsDefault: isDefault,
		}
	}

	items := make([]pickerItem, 0, len(profiles)+2)
	if defaultIdx >= 0 {
		items = append(items, itemFor(profiles[defaultIdx], true))
	}
	for i, p := range profiles {
		if i == defaultIdx {
			continue
		}
		items = append(items, itemFor(p, false))
	}
	if includeExtras {
		items = append(items,
			pickerItem{Name: profilePickerCreateNewLabel, IsExtra: true, Extra: profilePickerCreateNew},
			pickerItem{Name: profilePickerEnterHostLabel, IsExtra: true, Extra: profilePickerEnterHost},
		)
	}
	return items
}

// pickAuthProfile shows the auth profile picker and returns the user's choice.
// When the result is profilePickerProfile, the second return value is the
// selected profile name. For the other results it is empty.
func pickAuthProfile(ctx context.Context, profiles profile.Profiles, opts profilePickerOptions) (profilePickerResult, string, error) {
	items := buildPickerItems(profiles, opts.Default, opts.IncludeExtras)

	idx, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label:             opts.Label,
		Items:             items,
		StartInSearchMode: len(profiles) > 5,
		Searcher: func(input string, index int) bool {
			input = strings.ToLower(input)
			return strings.Contains(strings.ToLower(items[index].Name), input) ||
				strings.Contains(strings.ToLower(items[index].Host), input) ||
				strings.Contains(strings.ToLower(items[index].AccountID), input)
		},
		LabelTemplate: "{{ . | faint }}",
		Active:        `▸ {{.Name | bold}}{{if .IsDefault}} {{ "[default]" | green }}{{end}}{{if .AccountID}} (account: {{.AccountID|faint}}){{else if .Host}} ({{.Host|faint}}){{end}}`,
		Inactive:      `  {{.Name}}{{if .IsDefault}} [default]{{end}}{{if .AccountID}} (account: {{.AccountID|faint}}){{else if .Host}} ({{.Host|faint}}){{end}}`,
		Selected:      `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
	})
	if err != nil {
		return 0, "", err
	}

	picked := items[idx]
	if picked.IsExtra {
		return picked.Extra, "", nil
	}
	return profilePickerProfile, picked.Name, nil
}
