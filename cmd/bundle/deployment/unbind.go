package deployment

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/spf13/cobra"
)

func newUnbindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unbind KEY",
		Short:   "Unbind bundle-defined resources from its managed remote resource",
		Args:    cobra.ExactArgs(1),
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	var forceLock bool
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		r := b.Config.Resources
		resource, err := r.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		b.Config.Bundle.Deployment.Lock.Force = forceLock
		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Unbind(resource.TerraformResourceName(), args[0]),
		))
	}

	return cmd
}
