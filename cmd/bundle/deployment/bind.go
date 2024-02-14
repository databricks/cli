package deployment

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bind KEY RESOURCE_ID",
		Short:   "Bind bundle-defined resources to existing resources",
		Args:    cobra.ExactArgs(2),
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		r := b.Config.Resources
		resource, err := r.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()
		ctx := cmd.Context()
		exists, err := resource.Exists(ctx, w, args[1])
		if err != nil {
			return fmt.Errorf("failed to fetch the resource, err: %w", err)
		}

		if !exists {
			return fmt.Errorf("%s with an id '%s' is not found", resource.TerraformResourceName(), args[1])
		}

		b.Config.Bundle.Deployment.Lock.Force = forceLock
		err = bundle.Apply(cmd.Context(), b, bundle.Seq(
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
