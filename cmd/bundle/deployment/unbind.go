package deployment

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newUnbindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbind KEY",
		Short: "Unbind bundle-defined resources from its managed remote resource",
		Args:  root.ExactArgs(1),
	}

	var forceLock bool
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		resource, err := b.Config.Resources.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
			b.Config.Bundle.Deployment.Lock.Force = forceLock
		})

		tfName := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
		phases.Unbind(ctx, b, tfName, args[0])
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}
		return nil
	}

	return cmd
}
