package phases

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func assertRootPathExists(ctx context.Context, b *bundle.Bundle) (bool, error) {
	w := b.WorkspaceClient()
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.RootPath)

	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		log.Infof(ctx, "Root path does not exist: %s", b.Config.Workspace.RootPath)
		return false, nil
	}

	return true, err
}

func approvalForDestroy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	deleteActions := plan.GetActions()

	err := checkForPreventDestroy(b, deleteActions)
	if err != nil {
		return false, err
	}

	if len(deleteActions) > 0 {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
			if a.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	schemaActions := filterGroup(deleteActions, "schemas", deployplan.ActionTypeDelete)
	dltActions := filterGroup(deleteActions, "pipelines", deployplan.ActionTypeDelete)
	volumeActions := filterGroup(deleteActions, "volumes", deployplan.ActionTypeDelete)

	if len(schemaActions) > 0 {
		cmdio.LogString(ctx, deleteSchemaMessage)
		for _, a := range schemaActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(dltActions) > 0 {
		cmdio.LogString(ctx, deleteDltMessage)
		for _, a := range dltActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(volumeActions) > 0 {
		cmdio.LogString(ctx, deleteVolumeMessage)
		for _, a := range volumeActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	cmdio.LogString(ctx, "All files and directories at the following location will be deleted: "+b.Config.Workspace.RootPath)
	cmdio.LogString(ctx, "")

	if b.AutoApprove {
		return true, nil
	}

	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}

	return approved, nil
}

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType) {
	if engine.IsDirect() {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan, direct.MigrateMode(false))
	} else {
		// Core destructive mutators for destroy. These require informed user consent.
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, files.Delete())

	if !logdiag.HasError(ctx) {
		cmdio.LogString(ctx, "Destroy complete!")
	}
}

// The destroy phase deletes artifacts and resources.
func Destroy(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) {
	log.Info(ctx, "Phase: destroy")

	ok, err := assertRootPathExists(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if !ok {
		cmdio.LogString(ctx, "No active deployment found to destroy!")
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDestroy))
	}()

	if !engine.IsDirect() {
		bundle.ApplySeqContext(ctx, b,
			// We need to resolve artifact variable (how we do it in build phase)
			// because some of the to-be-destroyed resource might use this variable.
			// Not resolving might lead to terraform "Reference to undeclared resource" error
			mutator.ResolveVariableReferencesWithoutResources("artifacts"),
			mutator.ResolveVariableReferencesOnlyResources("artifacts"),

			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("destroy")),
		)
	}

	if logdiag.HasError(ctx) {
		return
	}

	var plan *deployplan.Plan
	if engine.IsDirect() {
		_, localPath := b.StateFilenameDirect(ctx)
		plan, err = b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(), nil, localPath)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		tf := b.Terraform
		if tf == nil {
			logdiag.LogError(ctx, errors.New("terraform not initialized"))
			return
		}

		plan, err = terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	hasApproval, err := approvalForDestroy(ctx, b, plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if hasApproval {
		destroyCore(ctx, b, plan, engine)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
