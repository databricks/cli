package skills

import (
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <words...>",
		Short: "Search the skills for words",
		Args:  cobra.MinimumNArgs(1),
		RunE:  notImplemented,
	}
}
