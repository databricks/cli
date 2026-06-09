package genie

import "github.com/spf13/cobra"

// NewGenieCmd creates the parent "genie" command group.
func NewGenieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "genie",
		Short:  "Ask data questions via Databricks Genie",
		Hidden: true,
	}

	cmd.AddCommand(newAskCmd())

	return cmd
}
