package aircmd

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get JOB_RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show details for a run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("get")
		},
	}

	return cmd
}
