package skills

import (
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the available skills",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}
