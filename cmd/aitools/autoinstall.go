package aitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
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
installation, as 'databricks aitools update' does. It never runs in CI and only
touches the global scope.

It stays off after 'databricks aitools auto-install disable' or a global-scope
'databricks aitools uninstall', until 'databricks aitools auto-install enable'.
Setting ` + autoinstall.EnvVar + `=false turns it off in that environment.`,
	}
	cmd.AddCommand(newAutoInstallToggleCmd(true))
	cmd.AddCommand(newAutoInstallToggleCmd(false))
	return cmd
}

// newAutoInstallToggleCmd returns `auto-install enable` or `auto-install disable`.
func newAutoInstallToggleCmd(enable bool) *cobra.Command {
	use, short := "disable", "Turn off the automatic installation of Databricks AI tools"
	if enable {
		use, short = "enable", "Turn the automatic installation of Databricks AI tools back on"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := autoinstall.SetOptedOut(ctx, !enable); err != nil {
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
	if autoinstall.EnvOptedOut(ctx) {
		cmdio.LogString(ctx, cmdio.Yellow(ctx, autoinstall.EnvVar+" still disables it in this environment; unset it to let it run."))
	}
}
