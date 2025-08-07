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
		Long: `Unbind a bundle resource from its linked workspace resource.

This command removes the link between a bundle definition and its corresponding
workspace resource. After unbinding, the workspace resource will no longer be
managed by the bundle and can be modified independently.

Arguments:
  KEY - The resource key defined in your bundle configuration to unbind

Examples:
  # Unbind a job resource
  databricks bundle deployment unbind my_etl_job

  # Unbind a pipeline resource
  databricks bundle deployment unbind data_pipeline

When to unbind:
- You want to stop managing a resource through the bundle
- You need to transfer resource ownership to manual management
- You're moving your resource to a different bundle or target

After unbinding:
- The workspace resource continues to exist and function normally
- Future bundle deployments will not affect the unbound resource
- You can manually modify the resource in the workspace UI
- The resource configuration remains in your bundle but does not have an
  associated workspace resource. On the next deployment a new copy will be
  created unless the configuration is removed.

To re-bind the resource later, use:
  databricks bundle deployment bind <KEY> <RESOURCE_ID>`,
		Args: root.ExactArgs(1),
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
