package ai

import (
	"github.com/spf13/cobra"
)

func newCancelCommand() *cobra.Command {
	var (
		all bool
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "cancel [RUN_ID...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Cancel one or more runs",
		Long:  `Cancel one or more runs by ID, or cancel all of your active runs with --all.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("cancel")
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Cancel all of your active runs")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")

	return cmd
}
