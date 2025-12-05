package cache

import (
	"github.com/databricks/cli/libs/cache"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Local cache related commands",
		Long:  "Manage local cache used by the Databricks CLI for improved performance",
	}

	cmd.AddCommand(newClearCommand())
	return cmd
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
