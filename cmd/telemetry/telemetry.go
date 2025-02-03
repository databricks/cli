package telemetry

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "telemetry",
		Short:  "CLI commands to publish telemetry to the Databricks backend.",
		Hidden: true,
	}

	cmd.AddCommand(newDummyCommand())
	return cmd
}
