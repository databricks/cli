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
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind KEY RESOURCE_ID",
		Short: "Bind bundle-defined resources to existing resources",
		Args:  root.ExactArgs(2),
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")
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

		w := b.WorkspaceClient()
		exists, err := resource.Exists(ctx, w, args[1])
		if err != nil {
			return fmt.Errorf("failed to fetch the resource, err: %w", err)
		}

		if !exists {
			return fmt.Errorf("%s with an id '%s' is not found", resource.ResourceDescription().SingularName, args[1])
		}

		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
			b.Config.Bundle.Deployment.Lock.Force = forceLock
		})

		tfName := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
		phases.Bind(ctx, b, &terraform.BindOptions{
			AutoApprove:  autoApprove,
			ResourceType: tfName,
			ResourceKey:  args[0],
			ResourceId:   args[1],
		})
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		cmdio.LogString(ctx, fmt.Sprintf("Successfully bound %s with an id '%s'. Run 'bundle deploy' to deploy changes to your workspace", resource.ResourceDescription().SingularName, args[1]))
		return nil
	}

	return cmd
}
