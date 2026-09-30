package deployment

import (
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
func BindResource(cmd *cobra.Command, resourceKey, resourceId string, autoApprove, forceLock, skipInitContext bool) error {
	_, err := bindResource(cmd, resourceKey, resourceId, autoApprove, forceLock, skipInitContext)
	return err
}

// bindResource reports whether the bind applied its changes immediately.
func bindResource(cmd *cobra.Command, resourceKey, resourceId string, autoApprove, forceLock, skipInitContext bool) (bool, error) {
	b, stateDesc, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{
		SkipInitContext: skipInitContext,
		AlwaysPull:      true,
		InitFunc: func(b *bundle.Bundle) {
			utils.SetForceLock(cmd, b, forceLock)
		},
	})
	if err != nil {
		return false, err
	}
	ctx := cmd.Context()

	resource, err := b.Config.Resources.FindResourceByConfigKey(resourceKey)
	if err != nil {
		return false, err
	}

	w := b.WorkspaceClient(ctx)
	exists, err := resource.Exists(ctx, w, resourceId)
	if err != nil {
		return false, fmt.Errorf("failed to fetch the resource, err: %w", err)
	}

	if !exists {
		return false, fmt.Errorf("%s with an id '%s' is not found", resource.ResourceDescription().SingularName, resourceId)
	}

	tfName, ok := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
	if !ok {
		tfName = resource.ResourceDescription().PluralName
	}
	if stateDesc.IsDMS() {
		if err := utils.OpenDirectStateForRead(ctx, b, stateDesc); err != nil {
			return false, err
		}
		defer func() {
			if _, err := b.DeploymentBundle.StateDB.Finalize(ctx); err != nil {
				logdiag.LogError(ctx, err)
			}
		}()
	}

	phases.Bind(ctx, b, &terraform.BindOptions{
		AutoApprove:  autoApprove,
		ResourceType: tfName,
		ResourceKey:  resourceKey,
		ResourceId:   resourceId,
	}, stateDesc)
	if logdiag.HasError(ctx) {
		return false, root.ErrAlreadyPrinted
	}

	cmdio.LogString(ctx, fmt.Sprintf("Successfully bound %s with an id '%s'", resource.ResourceDescription().SingularName, resourceId))
	return stateDesc.IsDMS(), nil
}
