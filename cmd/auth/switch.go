package auth

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

func newSwitchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Set the default profile",
		Long: `Set a named profile as the default in ~/.databrickscfg.

The selected profile name is stored in a [__settings__] section
in the config file under the default_profile key. Use "databricks auth profiles"
to see which profile is currently the default.`,
		Args: cobra.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configFile := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

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

			currentDefault, _ := databrickscfg.GetDefaultProfile(ctx, configFile)
			label := "Select a profile to set as default"
			if currentDefault != "" {
				label = fmt.Sprintf("Current default: %s. Select a new default", currentDefault)
			}
			result, selected, err := pickAuthProfile(ctx, allProfiles, profilePickerOptions{
				Label:   label,
				Default: currentDefault,
			})
			if err != nil {
				return err
			}
			// Without IncludeExtras, the picker only returns profile selections.
			if result != profilePickerProfile {
				return fmt.Errorf("unexpected picker result: %v", result)
			}
			profileName = selected
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
