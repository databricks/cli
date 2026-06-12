package aircmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// New returns the root command for the experimental AI runtime CLI.
//
// Milestone 0: scaffolds the command group with every subcommand registered as a
// stub (not yet implemented), pending the port from the Python `air` CLI.
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

// notImplemented returns the placeholder error used by milestone-0 stubs.
func notImplemented(name string) error {
	return fmt.Errorf("`air %s` is not implemented yet", name)
}
