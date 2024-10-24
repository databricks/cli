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
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sync"
	terraformlib "github.com/databricks/cli/libs/terraform"
	tfjson "github.com/hashicorp/terraform-json"
)

func parseTerraformActions(changes []*tfjson.ResourceChange, toInclude func(typ string, actions tfjson.Actions) bool) []terraformlib.Action {
	res := make([]terraformlib.Action, 0)
	for _, rc := range changes {
		if !toInclude(rc.Type, rc.Change.Actions) {
			continue
		}

		var actionType terraformlib.ActionType
		switch {
		case rc.Change.Actions.Delete():
			actionType = terraformlib.ActionTypeDelete
		case rc.Change.Actions.Replace():
			actionType = terraformlib.ActionTypeRecreate
		default:
			// No use case for other action types yet.
			continue
		}

		res = append(res, terraformlib.Action{
			Action:       actionType,
			ResourceType: rc.Type,
			ResourceName: rc.Name,
		})
	}

	return res
}

func approvalForDeploy(ctx context.Context, b *bundle.Bundle) (bool, error) {
	tf := b.Terraform
	if tf == nil {
		return false, fmt.Errorf("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return false, err
	}

	schemaActions := parseTerraformActions(plan.ResourceChanges, func(typ string, actions tfjson.Actions) bool {
		// Filter in only UC schema resources.
		if typ != "databricks_schema" {
			return false
		}

		// We only display prompts for destructive actions like deleting or
		// recreating a schema.
		return actions.Delete() || actions.Replace()
	})

	dltActions := parseTerraformActions(plan.ResourceChanges, func(typ string, actions tfjson.Actions) bool {
		// Filter in only DLT pipeline resources.
		if typ != "databricks_pipeline" {
			return false
		}

		// Recreating DLT pipeline leads to metadata loss and for a transient period
		// the underling tables will be unavailable.
		return actions.Replace() || actions.Delete()
	})

	// We don't need to display any prompts in this case.
	if len(dltActions) == 0 && len(schemaActions) == 0 {
		return true, nil
	}

	// One or more UC schema resources will be deleted or recreated.
	if len(schemaActions) != 0 {
		cmdio.LogString(ctx, "The following UC schemas will be deleted or recreated. Any underlying data may be lost:")
		for _, action := range schemaActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more DLT pipelines is being recreated.
	if len(dltActions) != 0 {
		msg := `
This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:`
		cmdio.LogString(ctx, msg)
		for _, action := range dltActions {
			cmdio.Log(ctx, action)
		}
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
				terraform.CheckDashboardsModifiedRemotely(),
				deploy.StatePull(),
				mutator.ValidateGitDetails(),
				artifacts.CleanUp(),
				libraries.ExpandGlobReferences(),
				libraries.Upload(),
				trampoline.TransformWheelTask(),
				files.Upload(outputHandler),
				deploy.StateUpdate(),
				deploy.StatePush(),
				permissions.ApplyWorkspaceRootPermissions(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.CheckRunningResource(),
				terraform.Plan(terraform.PlanGoal("deploy")),
				bundle.If(
					approvalForDeploy,
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
