package ai

import (
	"fmt"

	"github.com/spf13/cobra"
)

// New returns the root command for the experimental AI runtime CLI.
//
// Milestone 0: all subcommands are registered so the tree is navigable, but only
// `version` is implemented; the rest are stubs pending the port from Python `air`.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Run and manage AI runtime training workloads",
		Long: `Run and manage AI runtime training workloads on Databricks serverless GPU compute.

This command set is the Go port of the standalone Python "air" CLI. It is
experimental and may change in future versions.`,
	}

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newLogsCommand())
	cmd.AddCommand(newCancelCommand())
	cmd.AddCommand(newRegisterImageCommand())

	return cmd
}

// notImplemented returns the placeholder error used by milestone-0 stubs.
func notImplemented(name string) error {
	return fmt.Errorf("`ai %s` is not implemented yet", name)
}
