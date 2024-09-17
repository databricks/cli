package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/python"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sync"
	terraformlib "github.com/databricks/cli/libs/terraform"
)

func approvalForUcSchemaDelete(ctx context.Context, b *bundle.Bundle) (bool, error) {
	tf := b.Terraform
	if tf == nil {
		return false, fmt.Errorf("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return false, err
	}

	actions := make([]terraformlib.Action, 0)
	for _, rc := range plan.ResourceChanges {
		// We only care about destructive actions on UC schema resources.
		if rc.Type != "databricks_schema" {
			continue
		}

		var actionType terraformlib.ActionType

		switch {
		case rc.Change.Actions.Delete():
			actionType = terraformlib.ActionTypeDelete
		case rc.Change.Actions.Replace():
			actionType = terraformlib.ActionTypeRecreate
		default:
			// We don't need a prompt for non-destructive actions like creating
			// or updating a schema.
			continue
		}

		actions = append(actions, terraformlib.Action{
			Action:       actionType,
			ResourceType: rc.Type,
			ResourceName: rc.Name,
		})
	}

	// No restricted actions planned. No need for approval.
	if len(actions) == 0 {
		return true, nil
	}

	cmdio.LogString(ctx, "The following UC schemas will be deleted or recreated. Any underlying data may be lost:")
	for _, action := range actions {
		cmdio.Log(ctx, action)
	}

	if b.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, fmt.Errorf("the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
	}

	cmdio.LogString(ctx, "")
	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}

	return approved, nil
}

// The deploy phase deploys artifacts and resources.
func Deploy(outputHandler sync.OutputHandler) bundle.Mutator {
	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	deployCore := bundle.Defer(
		bundle.Seq(
			bundle.LogString("Deploying resources..."),
			terraform.Apply(),
		),
		bundle.Seq(
			terraform.StatePush(),
			terraform.Load(),
			metadata.Compute(),
			metadata.Upload(),
			bundle.LogString("Deployment complete!"),
		),
	)

	deployMutator := bundle.Seq(
		scripts.Execute(config.ScriptPreDeploy),
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				terraform.StatePull(),
				deploy.StatePull(),
				mutator.ValidateGitDetails(),
				artifacts.CleanUp(),
				libraries.ExpandGlobReferences(),
				libraries.Upload(),
				python.TransformWheelTask(),
				files.Upload(outputHandler),
				deploy.StateUpdate(),
				deploy.StatePush(),
				permissions.ApplyWorkspaceRootPermissions(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.CheckRunningResource(),
				terraform.Plan(terraform.PlanGoal("deploy")),
				bundle.If(
					approvalForUcSchemaDelete,
					deployCore,
					bundle.LogString("Deployment cancelled!"),
				),
			),
			lock.Release(lock.GoalDeploy),
		),
		scripts.Execute(config.ScriptPostDeploy),
	)

	return newPhase(
		"deploy",
		[]bundle.Mutator{deployMutator},
	)
}
