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
			// TODO: context can be on both command and feature level
			err := root.MustWorkspaceClient(cmd, args)
			if err != nil {
				return err
			}
			// TODO: add account-level init as well
			w := root.WorkspaceClient(cmd.Context())
			propagateEnvConfig(w.Config)

			state, err := feature.NewFeature(args[0])
			if err != nil {
				return err
			}
			return state.Install(ctx)
		},
	}
}
