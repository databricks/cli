package completion

import (
	"fmt"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var shellFlag string
	cmd := &cobra.Command{
		Use:               "install",
		Short:             "Install shell completions",
		Long:              "Install Databricks CLI tab completions into your shell configuration file.",
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

			filePath, alreadyInstalled, err := libcompletion.Install(shell, home)
			if err != nil {
				return err
			}

			if alreadyInstalled {
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions are already installed for %s in %s.", shell, filePath))
				return nil
			}

			cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions installed for %s.\nRestart your shell or run 'source %s' to activate.", shell, filePath))
			return nil
		},
	}
	addShellFlag(cmd, &shellFlag)
	return cmd
}
