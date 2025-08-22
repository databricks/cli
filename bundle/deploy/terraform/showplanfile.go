package terraform

import (
	"context"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func isSilentlyUpdated(resourceType string) bool {
	// These types are automatically created by DABs, no need to show them in the plan
	var silentlyUpdatedResources = []string{
		"databricks_grant",
		"databricks_permissions",
		"databricks_secret_acl",
	}

	for _, s := range silentlyUpdatedResources {
		if s == resourceType {
			return true
		}
	}
	return false
}

// GetActions converts Terraform resource changes into deployplan.Action values.
// The returned slice can be filtered using deployplan.Filter and FilterGroup helpers.
func GetActions(ctx context.Context, changes []*tfjson.ResourceChange) []deployplan.Action {
	var result []deployplan.Action

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
			if !isSilentlyUpdated(rc.Type) {
				log.Warnf(ctx, "unknown resource type '%s'", rc.Type)
			}
			continue
		}

		result = append(result, deployplan.Action{
			ActionType: actionType,
			ResourceNode: deployplan.ResourceNode{
				Group: group,
				Key:   rc.Name,
			},
		})
	}

	return result
}

// ShowPlanFile reads a Terraform plan file located at planPath using the provided tfexec.Terraform handle
// and converts it into a slice of deployplan.Action.
func ShowPlanFile(ctx context.Context, tf *tfexec.Terraform, planPath string) ([]deployplan.Action, error) {
	plan, err := tf.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, err
	}

	return GetActions(ctx, plan.ResourceChanges), nil
}
