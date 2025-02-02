package segment

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	terraformlib "github.com/databricks/cli/libs/terraform"
	tfjson "github.com/hashicorp/terraform-json"
)

func filterChangeActions(changes []*tfjson.ResourceChange, resourceType string) []terraformlib.Action {
	res := make([]terraformlib.Action, 0)
	for _, rc := range changes {
		if rc.Type != resourceType {
			continue
		}

		var actionType terraformlib.ActionType
		switch {
		case rc.Change.Actions.Create():
			actionType = terraformlib.ActionTypeCreate
		case rc.Change.Actions.Update():
			actionType = terraformlib.ActionTypeUpdate
		case rc.Change.Actions.Delete():
			actionType = terraformlib.ActionTypeDelete
		case rc.Change.Actions.Replace():
			actionType = terraformlib.ActionTypeRecreate
		default:
			// Filter other action types..
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

func ApprovalForDeploy(ctx context.Context, b *bundle.Bundle) (bool, error) {
	tf := b.Terraform
	if tf == nil {
		return false, errors.New("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return false, err
	}

	noteworthyActions := []terraformlib.ActionType{
		terraformlib.ActionTypeDelete,
		terraformlib.ActionTypeRecreate,
		terraformlib.ActionTypeCreate,
		terraformlib.ActionTypeUpdate,
	}

	if b.Config.Experimental.Segment.ConfirmAllChanges {
		noteworthyActions = append(noteworthyActions, []terraformlib.ActionType{}...)
	}

	jobActions := filterChangeActions(plan.ResourceChanges, "databricks_job")

	// We don't need to display any prompts in this case.
	if len(jobActions) == 0 {
		cmdio.LogString(ctx, "There are no changes detected in the plan, skipping user confirmation")
		return true, nil
	}

	if b.Config.Experimental.Segment.DetailedPlanView {
		// Define the command and arguments
		cmd := exec.Command("terraform", fmt.Sprintf("-chdir=%s", b.Terraform.WorkingDir()), "plan")

		// Run the command and capture the output
		output, err := cmd.Output()
		if err != nil {
			return false, err
		}

		cmdio.LogString(ctx, string(output))
	}

	// One or more DLT pipelines is being recreated.
	if len(jobActions) != 0 {
		cmdio.LogString(ctx, "This action will result in changes to the following Jobs")
		for _, action := range jobActions {
			cmdio.Log(ctx, action)
		}
	}

	if b.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("the deployment requires human review of planned actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
	}

	cmdio.LogString(ctx, "")
	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}

	return approved, nil
}
