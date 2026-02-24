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

func newUninstallCmd() *cobra.Command {
	var shellFlag string
	var autoApprove bool
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

			filePath := libcompletion.TargetFilePath(shell, home)
			displayPath := filepath.ToSlash(filePath)

			// Check current status to avoid a useless prompt.
			result, err := libcompletion.Status(shell, home)
			if err != nil {
				return err
			}

			if !result.Installed {
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions were not installed for %s.", shell))
				return nil
			}

			// Installed by another method (homebrew, package manager) — we can't uninstall it.
			if result.Method != "" && result.Method != "marker" {
				switch result.Method {
				case "homebrew":
					cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions for %s are provided by Homebrew. Nothing to uninstall.", shell))
				default:
					cmdio.LogString(ctx, fmt.Sprintf(
						"Databricks CLI completions for %s appear to be installed externally in %s. Nothing to uninstall.",
						shell,
						filepath.ToSlash(result.FilePath),
					))
				}
				warnIfCompinitMissing(ctx, shell, home)
				return nil
			}

			// Confirm before modifying.
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

			_, wasInstalled, err := libcompletion.Uninstall(shell, home)
			if err != nil {
				return err
			}

			if !wasInstalled {
				// Race: status said installed but uninstall found nothing.
				cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions were not installed for %s.", shell))
				return nil
			}

			cmdio.LogString(ctx, fmt.Sprintf("Databricks CLI completions removed for %s from %s.", shell, displayPath))
			return nil
		},
	}
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")
	addShellFlag(cmd, &shellFlag)
	return cmd
}
