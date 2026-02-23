package completion

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var shellFlag string
	var yes bool
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

			filePath := libcompletion.TargetFilePath(shell, home)
			displayPath := filepath.ToSlash(filePath)

			// Check if already installed — no confirmation needed.
			result, err := libcompletion.Status(shell, home)
			if err != nil {
				return err
			}
			if result.Installed && result.Method == "marker" {
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions are already installed for %s in %s.", shell, displayPath))
				return nil
			}

			// Confirm before writing.
			if !yes {
				if !cmdio.IsPromptSupported(ctx) {
					return errors.New("use --yes to skip the confirmation prompt in non-interactive mode")
				}
				cmdio.LogString(ctx, "Shell: "+shell.DisplayName())
				cmdio.LogString(ctx, "File:  "+displayPath)
				confirmed, err := cmdio.AskYesOrNo(ctx, "Proceed?")
				if err != nil {
					return err
				}
				if !confirmed {
					return nil
				}
			}

			_, _, err = libcompletion.Install(shell, home)
			if err != nil {
				return err
			}

			msg := fmt.Sprintf("Databricks CLI completions installed for %s.\n", shell)
			switch shell {
			case libcompletion.PowerShell, libcompletion.PowerShell5:
				msg += "Restart your shell to activate."
			default:
				msg += fmt.Sprintf("Restart your shell or run 'source %s' to activate.", displayPath)
			}
			cmdio.LogString(ctx, msg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	addShellFlag(cmd, &shellFlag)
	return cmd
}
