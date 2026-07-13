package aircmd

import "github.com/spf13/cobra"

// New creates the parent "air" command group for migrating AIR CLI workloads
// onto Databricks Asset Bundles.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "air",
		Short:  "Migrate AI Runtime (AIR) CLI workloads to Databricks Asset Bundles",
		Hidden: true,
	}

	cmd.AddCommand(newInitCommand())

	return cmd
}
