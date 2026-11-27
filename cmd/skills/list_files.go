package skills

import (
	"github.com/spf13/cobra"
)

func newListFilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-files <name>",
		Short: "List the files of a skill",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented,
	}
}
