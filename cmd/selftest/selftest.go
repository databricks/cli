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

	cmd.AddCommand(newChildCommand())
	cmd.AddCommand(newParentCommand())
	cmd.AddCommand(newSendTelemetry())
	return cmd
}
