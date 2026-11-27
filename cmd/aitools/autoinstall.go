package aitools

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

func newAutoInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-install",
		Short: "Turn the automatic installation of Databricks AI tools on or off",
		Long: `Turn the automatic installation of Databricks AI tools on or off.

On the first run of a new CLI version, the CLI can install the Databricks AI
tools (skills and plugins) for the coding agents already on this machine, as
'databricks aitools install --scope=global' does, or update an existing
installation, as 'databricks aitools update' does. It never runs in CI or on
Databricks Runtime, only touches the global scope, and asks nothing.

It runs only once the user agreed to it with 'databricks aitools auto-install
enable', a global-scope 'databricks aitools install' or a "yes" to the skills
prompt of 'databricks apps init', or when it finds a global installation that
the CLI made.
It stays off after 'databricks aitools auto-install disable', a global-scope
'databricks aitools uninstall', or a "no" to the skills prompt of 'databricks
apps init', until 'databricks aitools auto-install enable'. Setting
` + autoinstall.EnvVar + ` to anything but a true value (true, 1, yes, on) turns
it off in that environment. Setting ` + skills.AgentFeaturesDisabledEnv + `
to a true value (true, 1, yes, on) also turns it off, along with the skills and
the aitools hints.`,
		RunE: root.ReportUnknownSubcommand,
	}
	cmd.AddCommand(newAutoInstallToggleCmd(true))
	cmd.AddCommand(newAutoInstallToggleCmd(false))
	return cmd
}

// newAutoInstallToggleCmd returns `auto-install enable` or `auto-install disable`.
func newAutoInstallToggleCmd(enable bool) *cobra.Command {
	use, short := "disable", "Turn off the automatic installation of Databricks AI tools"
	if enable {
		use, short = "enable", "Turn on the automatic installation of Databricks AI tools"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consent := autoinstall.ConsentOptedOut
			if enable {
				consent = autoinstall.ConsentGiven
			}
			if _, err := autoinstall.SetConsent(ctx, consent); err != nil {
				return fmt.Errorf("failed to save the auto-install setting: %w", err)
			}
			printAutoInstallSetting(ctx, enable)
			return nil
		},
	}
}

// printAutoInstallSetting confirms the stored setting. An enable that the
// environment still overrides says so, rather than looking like it worked.
func printAutoInstallSetting(ctx context.Context, enabled bool) {
	if !enabled {
		cmdio.LogString(ctx, "Automatic installation of Databricks AI tools is disabled.")
		return
	}
	cmdio.LogString(ctx, "Automatic installation of Databricks AI tools is enabled.")
	// Both variables can be set. They share one warning, so unsetting only one
	// is not mistaken for a fix.
	var overrides []string
	if autoinstall.EnvOptedOut(ctx) {
		overrides = append(overrides, autoinstall.EnvVar)
	}
	if skills.AgentFeaturesDisabled(ctx) {
		overrides = append(overrides, skills.AgentFeaturesDisabledEnv)
	}
	switch len(overrides) {
	case 0:
	case 1:
		cmdio.LogString(ctx, cmdio.Yellow(ctx, overrides[0]+" still disables it in this environment; unset it to let it run."))
	default:
		cmdio.LogString(ctx, cmdio.Yellow(ctx, strings.Join(overrides, " and ")+" still disable it in this environment; unset both to let it run."))
	}
}
