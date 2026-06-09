package ai

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var (
		limit    int
		active   bool
		allUsers bool
		filters  []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Args:  root.NoArgs,
		Short: "List recent runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("list")
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&active, "active", false, "Show only active runs")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show runs from all users")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter runs, e.g. experiment=foo* (repeatable)")

	return cmd
}
