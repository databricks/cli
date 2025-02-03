package telemetry

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "telemetry",
		Short:  "",
		Hidden: true,
	}

	cmd.AddCommand(newDummyCommand())
	return cmd
}
