package completion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var shellFlag string
	cmd := &cobra.Command{
		Use:               "uninstall",
		Short:             "Uninstall shell completions",
		Long:              "Remove Databricks CLI tab completions from your shell configuration file.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			shell, err := libcompletion.DetectShell(shellFlag)
			if err != nil {
				return err
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			filePath, wasInstalled, err := libcompletion.Uninstall(shell, home)
			if err != nil {
				return err
			}
			displayPath := filepath.ToSlash(filePath)

			if !wasInstalled {
				result, statusErr := libcompletion.Status(shell, home)
				if statusErr == nil && result.Installed && result.Method != "" && result.Method != "marker" {
					cmdio.LogString(ctx, fmt.Sprintf(
						"Databricks CLI completions for %s appear to be installed via %s in %s. Nothing to uninstall.",
						shell,
						result.Method,
						filepath.ToSlash(result.FilePath),
					))
					return nil
				}
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions were not installed for %s.", shell))
				return nil
			}

			cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions removed for %s from %s.", shell, displayPath))
			return nil
		},
	}
	addShellFlag(cmd, &shellFlag)
	return cmd
}
