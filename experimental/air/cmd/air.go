package aircmd

import (
	"github.com/spf13/cobra"
)

// New returns the root command for the experimental AI runtime CLI.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "air",
		Short: "Run and manage AI runtime training workloads",
		Long: `Run and manage AI runtime training workloads on Databricks serverless GPU compute.

This command set is the Go port of the standalone Python "air" CLI. It is
experimental and may change in future versions.`,
	}

	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newLogsCommand())
	cmd.AddCommand(newCancelCommand())
	cmd.AddCommand(newRegisterImageCommand())

	return cmd
}
