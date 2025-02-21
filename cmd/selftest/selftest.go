package selftest

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "selftest",
		Short:  "Non functional CLI commands that are useful for testing",
		Hidden: true,
	}

	// TODO: Run the acceptance tests as integration tests?
	cmd.AddCommand(newSendTelemetry())
	return cmd
}
