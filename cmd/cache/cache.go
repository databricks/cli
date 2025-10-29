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
		Long:  "Remove all cached files stored locally by the Databricks CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return cache.ClearFileCache(ctx)
		},
	}
	return cmd
}
