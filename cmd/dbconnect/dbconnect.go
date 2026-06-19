package dbconnect

import "github.com/spf13/cobra"

// New returns the `dbconnect` command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dbconnect",
		Short:   "Set up a local Python environment matched to your Databricks compute",
		GroupID: "development",
		Long: `Set up a local Python environment matched to your Databricks compute target.

Derives the Python version, databricks-connect version, and dependency
constraints from the selected compute (cluster, serverless, or job) so that
local resolution matches the Databricks runtime.`,
	}
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newSyncCommand())
	return cmd
}
