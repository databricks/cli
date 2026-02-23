package completion

import (
	"fmt"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var shellFlag string
	cmd := &cobra.Command{
		Use:               "status",
		Short:             "Show shell completion status",
		Long:              "Show whether Databricks CLI tab completions are installed for your shell.",
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

			result, err := libcompletion.Status(shell, home)
			if err != nil {
				return err
			}

			statusStr := "not installed"
			if result.Installed {
				statusStr = "installed"
				if result.Method != "" && result.Method != "marker" {
					statusStr = fmt.Sprintf("installed (via %s)", result.Method)
				}
			}

			cmdio.LogString(ctx, "Shell:   "+shell.DisplayName())
			cmdio.LogString(ctx, "File:    "+result.FilePath)
			cmdio.LogString(ctx, "Status:  "+statusStr)
			return nil
		},
	}
	addShellFlag(cmd, &shellFlag)
	return cmd
}
