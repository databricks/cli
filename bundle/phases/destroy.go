package phases

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/bundle/terranova"
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

func getDeleteActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if b.DirectDeployment {
		return terranova.CalculateDestroyActions(ctx, b)
	}

	tf := b.Terraform

	if tf == nil {
		return nil, errors.New("terraform not initialized")
	}

	actions, err := terraform.ShowPlanFile(ctx, tf, b.Plan.TerraformPlanPath)
	if err != nil {
		return nil, err
	}

	deleteActions := deployplan.Filter(actions, deployplan.ActionTypeDelete)

	return deleteActions, nil
}

func approvalForDestroy(ctx context.Context, b *bundle.Bundle) (bool, error) {
	deleteActions, err := getDeleteActions(ctx, b)
	if err != nil {
		return false, err
	}

	b.Plan.Actions = deleteActions

	if len(deleteActions) > 0 {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
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

func destroyCore(ctx context.Context, b *bundle.Bundle) {
	if b.DirectDeployment {
		bundle.ApplyContext(ctx, b, terranova.TerranovaApply())
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
func Destroy(ctx context.Context, b *bundle.Bundle) {
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

	bundle.ApplyContext(ctx, b, statemgmt.StatePull())
	if logdiag.HasError(ctx) {
		return
	}

	if !b.DirectDeployment {
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("destroy")),
		)
	}

	if logdiag.HasError(ctx) {
		return
	}

	hasApproval, err := approvalForDestroy(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if hasApproval {
		destroyCore(ctx, b)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
