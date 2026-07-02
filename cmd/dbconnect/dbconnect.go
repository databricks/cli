package dbconnect

import (
	libsdbconnect "github.com/databricks/cli/libs/dbconnect"
	"github.com/spf13/cobra"
)

// New returns the dbconnect command group. The group and verb names come from
// the single command-name constants in libs/dbconnect so a rename is a
// one-location change (spec §0 / invariant 8).
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     libsdbconnect.CommandGroup,
		Short:   "Set up a local Python environment matched to your Databricks compute",
		GroupID: "development",
		Long: `Set up a local Python environment matched to your Databricks compute target.

Derives the Python version, databricks-connect version, and dependency
constraints from the selected compute (cluster, serverless, or job) so that
local resolution matches the Databricks runtime.`,
	}
	cmd.AddCommand(newSyncCommand())
	return cmd
}
