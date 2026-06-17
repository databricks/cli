package aircmd

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newCancelCommand() *cobra.Command {
	var (
		all bool
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "cancel [JOB_RUN_ID...]",
		Short: "Cancel one or more runs",
		Long:  `Cancel one or more runs by ID, or cancel all of your active runs with --all.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("cancel")
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Cancel all of your active runs")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")

	// Require exactly one of: one or more JOB_RUN_IDs, or --all. Cobra parses flags
	// before running this, so `all` reflects the user's input.
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		switch {
		case all && len(args) > 0:
			return &root.InvalidArgsError{Command: cmd, Message: "cannot combine JOB_RUN_ID arguments with --all"}
		case !all && len(args) == 0:
			return &root.InvalidArgsError{Command: cmd, Message: "provide at least one JOB_RUN_ID, or use --all"}
		}
		return nil
	}

	return cmd
}
