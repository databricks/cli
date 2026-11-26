package skills

import (
	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <name>",
		Short: "Read a skill's instructions",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented,
	}
}
