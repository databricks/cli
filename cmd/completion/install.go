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
	var autoApprove bool
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
			if result.Installed {
				switch result.Method {
				case "marker":
					// Our shim is already in the RC file — nothing to do.
					cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions are already installed for %s in %s.", shell, displayPath))
					warnIfCompinitMissing(ctx, shell, home)
					return nil
				case "homebrew":
					// Homebrew provides completions via a separate file. The user
					// may still want a CLI-managed shim in .zshrc (e.g. for a
					// newer binary). Inform them and proceed with install.
					cmdio.LogString(ctx, fmt.Sprintf("Note: Databricks CLI completions for %s are also provided by Homebrew.", shell))
				default:
					// External file (e.g. fish installed by package manager) — we
					// can't overwrite it, so report and exit.
					cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions for %s are already present in %s.", shell, displayPath))
					warnIfCompinitMissing(ctx, shell, home)
					return nil
				}
			}

			// Confirm before writing.
			if !autoApprove {
				if !cmdio.IsPromptSupported(ctx) {
					return errors.New("use --auto-approve to skip the confirmation prompt, or run 'databricks completion status' to preview the detected shell and target file")
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

			_, alreadyInstalled, err := libcompletion.Install(shell, home)
			if err != nil {
				return err
			}
			if alreadyInstalled {
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions are already installed for %s in %s.", shell, displayPath))
				return nil
			}

			msg := fmt.Sprintf("Databricks CLI completions installed for %s.\n", shell)
			switch shell {
			case libcompletion.PowerShell, libcompletion.PowerShell5:
				msg += "Restart your shell to activate."
			default:
				msg += fmt.Sprintf("Restart your shell or run 'source %s' to activate.", displayPath)
			}
			cmdio.LogString(ctx, msg)
			warnIfCompinitMissing(ctx, shell, home)
			return nil
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")
	addShellFlag(cmd, &shellFlag)
	return cmd
}
