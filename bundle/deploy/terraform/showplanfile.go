package terraform

import (
	"context"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// silentlyUpdatedResources contains resource types that are automatically created by DABs,
// no need to show them in the plan
var silentlyUpdatedResources = map[string]bool{
	"databricks_grants":      true,
	"databricks_permissions": true,
	"databricks_secret_acl":  true,
}

// populatePlan populates a deployplan.Plan from Terraform resource changes.
func populatePlan(ctx context.Context, plan *deployplan.Plan, changes []*tfjson.ResourceChange) {
	for _, rc := range changes {
		if rc.Change == nil {
			continue
		}

		var actionType deployplan.ActionType
		switch {
		case rc.Change.Actions.Delete():
			actionType = deployplan.ActionTypeDelete
		case rc.Change.Actions.Replace():
			actionType = deployplan.ActionTypeRecreate
		case rc.Change.Actions.Create():
			actionType = deployplan.ActionTypeCreate
		case rc.Change.Actions.Update():
			actionType = deployplan.ActionTypeUpdate
		default:
			continue
		}

		group, ok := TerraformToGroupName[rc.Type]
		if !ok {
			if !silentlyUpdatedResources[rc.Type] {
				log.Warnf(ctx, "unknown resource type '%s'", rc.Type)
			}
			continue
		}

		key := "resources." + group + "." + rc.Name
		plan.Plan[key] = &deployplan.PlanEntry{Action: actionType.String()}
	}
}

// ShowPlanFile reads a Terraform plan file located at planPath using the provided tfexec.Terraform handle
// and converts it into a deployplan.Plan.
func ShowPlanFile(ctx context.Context, tf *tfexec.Terraform, planPath string) (*deployplan.Plan, error) {
	tfPlan, err := tf.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, err
	}

	plan := &deployplan.Plan{
		Plan: make(map[string]*deployplan.PlanEntry),
	}

	populatePlan(ctx, plan, tfPlan.ResourceChanges)

	return plan, nil
}
