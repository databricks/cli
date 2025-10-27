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

// BindResource binds a bundle resource to an existing workspace resource.
// This function is shared between the bind command and generate commands with --bind flag.
func BindResource(cmd *cobra.Command, resourceKey, resourceId string, autoApprove, forceLock bool) error {
	ctx := cmd.Context()
	// Reload the bundle configuration to ensure we're using the latest configuration.
	b := utils.ConfigureBundleWithVariables(cmd)
	if b == nil || logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	resource, err := b.Config.Resources.FindResourceByConfigKey(resourceKey)
	if err != nil {
		return err
	}

	w := b.WorkspaceClient()
	exists, err := resource.Exists(ctx, w, resourceId)
	if err != nil {
		return fmt.Errorf("failed to fetch the resource, err: %w", err)
	}

	if !exists {
		return fmt.Errorf("%s with an id '%s' is not found", resource.ResourceDescription().SingularName, resourceId)
	}

	bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
		b.Config.Bundle.Deployment.Lock.Force = forceLock
	})

	tfName := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
	phases.Bind(ctx, b, &terraform.BindOptions{
		AutoApprove:  autoApprove,
		ResourceType: tfName,
		ResourceKey:  resourceKey,
		ResourceId:   resourceId,
	})
	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	cmdio.LogString(ctx, fmt.Sprintf("Successfully bound %s with an id '%s'", resource.ResourceDescription().SingularName, resourceId))
	return nil
}
