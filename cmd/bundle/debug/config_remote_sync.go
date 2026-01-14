package debug

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func NewConfigRemoteSyncCommand() *cobra.Command {
	var save bool

	cmd := &cobra.Command{
		Use:   "config-remote-sync",
		Short: "Sync remote resource changes to bundle configuration (experimental)",
		Long: `Compares deployed state with current remote state and generates updated configuration files.

When --save is specified, writes updated YAML files to disk.
Otherwise, outputs diff without modifying files.

Examples:
  # Show diff without saving
  databricks bundle debug config-remote-sync

  # Show diff and save to files
  databricks bundle debug config-remote-sync --save`,
		Hidden: true, // Used by DABs in the Workspace only
	}

	cmd.Flags().BoolVar(&save, "save", false, "Write updated config files to disk")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, _, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{})
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		changes, err := configsync.DetectChanges(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to detect changes: %w", err)
		}

		files, err := configsync.GenerateYAMLFiles(ctx, b, changes)
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
