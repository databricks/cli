package cache

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cache"
	"github.com/spf13/cobra"
)

// Spec declares the cache command.
var Spec = root.CommandSpec{
	Use:   "cache",
	Short: "Local cache related commands",
	Build: buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = "Manage local cache used by the Databricks CLI for improved performance"

	cmd.AddCommand(newClearCommand())
}

func newClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all local cache files",
		Long: `Remove all cached files stored locally by the Databricks CLI.

This clears the cache for all CLI versions, not just the current version.
The cache directory is typically located at:
  - Linux/macOS: ~/.cache/databricks/
  - Windows: %LOCALAPPDATA%\databricks\

You can override this with the DATABRICKS_CACHE_DIR environment variable.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cachePath, err := cache.ClearFileCache(ctx)
			if err != nil {
				return err
			}
			cmd.Printf("Cache cleared successfully from %s\n", cachePath)
			return nil
		},
	}
	return cmd
}
