package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func newSwitchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Set the default profile",
		Long: `Set a named profile as the default in ~/.databrickscfg.

The selected profile name is stored in a [databricks-cli-settings] section
in the config file under the default_profile key. Use "databricks auth profiles"
to see which profile is currently the default.`,
		Args: cobra.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configFile := os.Getenv("DATABRICKS_CONFIG_FILE")

		profileName := cmd.Flag("profile").Value.String()

		if profileName == "" {
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("the command is being run in a non-interactive environment, please specify a profile using --profile")
			}

			allProfiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
			if err != nil {
				return err
			}
			if len(allProfiles) == 0 {
				return errors.New("no profiles configured. Run 'databricks auth login' to create a profile")
			}

			// Use the already-loaded config file to resolve the current default,
			// avoiding a redundant file read.
			currentDefault := ""
			if iniFile, err := profile.DefaultProfiler.Get(ctx); err == nil {
				currentDefault = databrickscfg.GetDefaultProfileFrom(iniFile)
			}
			selectedName, err := promptForSwitchProfile(ctx, allProfiles, currentDefault)
			if err != nil {
				return err
			}
			profileName = selectedName
		} else {
			// Validate the profile exists.
			profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName(profileName))
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				return fmt.Errorf("profile %q not found", profileName)
			}
		}

		err := databrickscfg.SetDefaultProfile(ctx, profileName, configFile)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Default profile set to %q.", profileName))
		return nil
	}

	return cmd
}

// promptForSwitchProfile shows an interactive profile picker for the switch command.
// Reuses profileSelectItem from token.go for consistent display.
func promptForSwitchProfile(ctx context.Context, profiles profile.Profiles, currentDefault string) (string, error) {
	items := make([]profileSelectItem, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, profileSelectItem{Name: p.Name, Host: p.Host})
	}

	label := "Select a profile to set as default"
	if currentDefault != "" {
		label = fmt.Sprintf("Current default: %s. Select a new default", currentDefault)
	}

	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             label,
		Items:             items,
		StartInSearchMode: len(profiles) > 5,
		Searcher: func(input string, index int) bool {
			input = strings.ToLower(input)
			name := strings.ToLower(items[index].Name)
			host := strings.ToLower(items[index].Host)
			return strings.Contains(name, input) || strings.Contains(host, input)
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}}{{if .Host}} ({{.Host|faint}}){{end}}`,
			Inactive: `{{.Name}}{{if .Host}} ({{.Host}}){{end}}`,
			Selected: `{{ "Default profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}
