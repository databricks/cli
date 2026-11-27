package skills

import (
	"github.com/spf13/cobra"
)

func newReadFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read-file <name> <path>",
		Short: "Read one file of a skill",
		Args:  cobra.ExactArgs(2),
		RunE:  notImplemented,
	}
}
