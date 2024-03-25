package deployment

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bind KEY RESOURCE_ID",
		Short:   "Bind bundle-defined resources to existing resources",
		Args:    root.ExactArgs(2),
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)
		resource, err := b.Config.Resources.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()
		exists, err := resource.Exists(ctx, w, args[1])
		if err != nil {
			return fmt.Errorf("failed to fetch the resource, err: %w", err)
		}

		if !exists {
			return fmt.Errorf("%s with an id '%s' is not found", resource.TerraformResourceName(), args[1])
		}

		bundle.ApplyFunc(ctx, b, func(context.Context, *bundle.Bundle) error {
			b.Config.Bundle.Deployment.Lock.Force = forceLock
			return nil
		})

		err = bundle.Apply(ctx, b, bundle.Seq(
			phases.Initialize(),
			phases.Bind(&terraform.BindOptions{
				AutoApprove:  autoApprove,
				ResourceType: resource.TerraformResourceName(),
				ResourceKey:  args[0],
				ResourceId:   args[1],
			}),
		))
		if err != nil {
			return fmt.Errorf("failed to bind the resource, err: %w", err)
		}

		cmdio.LogString(ctx, fmt.Sprintf("Successfully bound %s with an id '%s'. Run 'bundle deploy' to deploy changes to your workspace", resource.TerraformResourceName(), args[1]))
		return nil
	}

	return cmd
}
