package telemetry

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "telemetry",
		Short:  "commands that are used to upload telemetry",
		Hidden: true,
	}

	cmd.AddCommand(newTelemetryUpload())
	return cmd
}
