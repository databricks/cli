package terraform

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// GetActions converts Terraform resource changes into deployplan.Action values.
// The returned slice can be filtered using deployplan.Filter and FilterGroup helpers.
func GetActions(changes []*tfjson.ResourceChange) []deployplan.Action {
	supported := config.SupportedResources()
	typeToGroup := make(map[string]string, len(supported))
	for group, desc := range supported {
		typeToGroup[desc.TerraformResourceName] = group
	}

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

		group, ok := typeToGroup[rc.Type]
		if !ok {
			// Happens for databricks_grant, databricks_permissions, databricks_secrets_acl.
			// These are automatically created by DABs, no need to show them.
			continue
		}

		result = append(result, deployplan.Action{
			Action: actionType,
			Group:  group,
			Name:   rc.Name,
		})
	}

	return result
}

// ShowPlanFile reads a Terraform plan file located at planPath using the provided tfexec.Terraform handle
// and converts it into a slice of deployplan.Action.
//
// The conversion maps Terraform resource types (e.g. "databricks_pipeline") to bundle configuration
// resource groups (e.g. "pipelines") using config.SupportedResources(). If a resource type is not
// recognised we fall back to the raw Terraform resource type so that the information is not lost.
func ShowPlanFile(ctx context.Context, tf *tfexec.Terraform, planPath string) ([]deployplan.Action, error) {
	plan, err := tf.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, err
	}

	return GetActions(plan.ResourceChanges), nil
}
