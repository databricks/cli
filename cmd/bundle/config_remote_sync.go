package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newConfigRemoteSyncCommand() *cobra.Command {
	var save bool

	cmd := &cobra.Command{
		Use:   "config-remote-sync",
		Short: "[EXPERIMENTAL] Sync remote resource changes to bundle configuration",
		Long: `[EXPERIMENTAL] Compares deployed state with current remote state and generates updated configuration files.

When --save is specified, writes updated YAML files to disk.
Otherwise, outputs diff without modifying files.

IMPORTANT: This is an experimental feature and is subject to change. Windows is not yet supported.

Examples:
  # Show diff without saving
  databricks bundle config-remote-sync

  # Show diff and save to files
  databricks bundle config-remote-sync --save`,
		Hidden: true, // Used by DABs in the Workspace only
	}

	cmd.Flags().BoolVar(&save, "save", false, "Write updated config files to disk")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS == "windows" {
			return errors.New("config-remote-sync command is not supported on Windows")
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{
			ReadState:  true,
			Build:      true,
			AlwaysPull: true,
		})
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		changes, err := configsync.DetectChanges(ctx, b, stateDesc.Engine)
		if err != nil {
			return fmt.Errorf("failed to detect changes: %w", err)
		}

		changesByFile, err := configsync.GetResolvedFieldChanges(ctx, b, changes)
		if err != nil {
			return fmt.Errorf("failed to resolve field changes: %w", err)
		}

		files, err := configsync.ApplyChangesToYAML(ctx, b, changesByFile)
		if err != nil {
			return fmt.Errorf("failed to generate YAML files: %w", err)
		}

		if save {
			if err := configsync.SaveFiles(ctx, b, files); err != nil {
				return fmt.Errorf("failed to save files: %w", err)
			}
		}

		var result []byte
		if root.OutputType(cmd) == flags.OutputJSON {
			diffOutput := &configsync.DiffOutput{
				Files:   files,
				Changes: changes,
			}
			result, err = json.MarshalIndent(diffOutput, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}
		} else if root.OutputType(cmd) == flags.OutputText {
			result = []byte(configsync.FormatTextOutput(changes))
		}

		out := cmd.OutOrStdout()
		_, _ = out.Write(result)
		_, _ = out.Write([]byte{'\n'})
		return nil
	}

	return cmd
}
