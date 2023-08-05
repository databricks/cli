package labs

import (
	"github.com/databricks/cli/cmd/labs/feature"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "install NAME",
		Short:   "Install a feature",
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			state, err := feature.NewFeature(args[0])
			if err != nil {
				return err
			}
			return state.Install(ctx)
		},
	}
}
