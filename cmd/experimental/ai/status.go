package ai

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show status and configuration for a run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("status")
		},
	}

	return cmd
}
