package phases

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"

	"github.com/databricks/cli/libs/cmdio"

	"github.com/databricks/cli/libs/log"
	terraformlib "github.com/databricks/cli/libs/terraform"
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

func approvalForDestroy(ctx context.Context, b *bundle.Bundle) (bool, error) {
	tf := b.Terraform
	if tf == nil {
		return false, errors.New("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return false, err
	}

	deleteActions := make([]terraformlib.Action, 0)
	for _, rc := range plan.ResourceChanges {
		if rc.Change.Actions.Delete() {
			deleteActions = append(deleteActions, terraformlib.Action{
				Action:       terraformlib.ActionTypeDelete,
				ResourceType: rc.Type,
				ResourceName: rc.Name,
			})
		}
	}

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

// The destroy phase deletes artifacts and resources.
func Destroy() bundle.Mutator {
	// Core destructive mutators for destroy. These require informed user consent.
	destroyCore := bundle.Seq(
		terraform.Apply(),
		files.Delete(),
		bundle.LogString("Destroy complete!"),
	)

	destroyMutator := bundle.Seq(
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				terraform.StatePull(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.Plan(terraform.PlanGoal("destroy")),
				bundle.If(
					approvalForDestroy,
					destroyCore,
					bundle.LogString("Destroy cancelled!"),
				),
			),
			lock.Release(lock.GoalDestroy),
		),
	)

	return newPhase(
		"destroy",
		[]bundle.Mutator{
			// Only run deploy mutator if root path exists.
			bundle.If(
				assertRootPathExists,
				destroyMutator,
				bundle.LogString("No active deployment found to destroy!"),
			),
		},
	)
}
