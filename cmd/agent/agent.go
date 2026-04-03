package agent

import (
	"github.com/spf13/cobra"
)

// New returns the agent command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "agent",
		Short:  "Commands for AI agent integration",
		Hidden: true,
	}

	cmd.AddCommand(newConsentCommand())

	return cmd
}
