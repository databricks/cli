package deployment

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
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
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = phases.Initialize(ctx, b)
		if err := diags.Error(); err != nil {
			return fmt.Errorf("failed to initialize bundle, err: %w", err)
		}

		resource, err := b.Config.Resources.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		bundle.ApplyFunc(ctx, b, func(context.Context, *bundle.Bundle) diag.Diagnostics {
			b.Config.Bundle.Deployment.Lock.Force = forceLock
			return nil
		})

		tfName := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
		diags = diags.Extend(phases.Unbind(ctx, b, tfName, args[0]))
		if err := diags.Error(); err != nil {
			return fmt.Errorf("failed to unbind the resource, err: %w", err)
		}
		return nil
	}

	return cmd
}
