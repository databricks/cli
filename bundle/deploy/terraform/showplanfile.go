package terraform

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// silentlyUpdatedResources contains resource types that are automatically created by DABs,
// no need to show them in the plan
var silentlyUpdatedResources = map[string]bool{
	"databricks_secret_acl": true,
}

var prefixToGroup = []struct{ prefix, group string }{
	{"job_", "jobs"},
	{"pipeline_", "pipelines"},
	{"mlflow_experiment_", "experiments"},
	{"mlflow_model_", "models"},
	{"cluster_", "clusters"},
	{"app_", "apps"},
	{"dashboard_", "dashboards"},
	{"alert_", "alerts"},
	{"model_serving_", "model_serving_endpoints"},
	{"sql_endpoint_", "sql_warehouses"},
	{"database_instance_", "database_instances"},
}

var grantsPrefix = []struct{ prefix, group string }{
	{"schema_", "schemas"},
	{"volume_", "volumes"},
	{"registered_model_", "registered_models"},
}

// convertPermissionResourceNameToKey converts terraform permission resource names back to hierarchical resource keys
// e.g., "mlflow_experiment_foo" -> "resources.experiments.foo.permissions"
func convertPermissionsResourceNameToKey(terraformName string) string {
	for _, pg := range prefixToGroup {
		if resourceName, found := strings.CutPrefix(terraformName, pg.prefix); found {
			return "resources." + pg.group + "." + resourceName + ".permissions"
		}
	}

	// Fallback: if no known prefix is found, use the old behavior
	return ""
}

// convertGrantsResourceNameToKey converts terraform grants resource names back to hierarchical resource keys
// e.g., "schema_foo" -> "resources.schemas.foo.grants"
func convertGrantsResourceNameToKey(terraformName string) string {
	for _, gp := range grantsPrefix {
		if resourceName, found := strings.CutPrefix(terraformName, gp.prefix); found {
			return "resources." + gp.group + "." + resourceName + ".grants"
		}
	}

	// Fallback: if no known prefix is found, use the old behavior
	return ""
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
		case rc.Change.Actions.NoOp():
			actionType = deployplan.ActionTypeSkip
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

		var key string
		switch group {
		case "permissions":
			// Convert terraform permission resource name back to hierarchical resource key
			key = convertPermissionsResourceNameToKey(rc.Name)
		case "grants":
			// Convert terraform grants resource name back to hierarchical resource key
			key = convertGrantsResourceNameToKey(rc.Name)
		default:
			key = "resources." + group + "." + rc.Name
		}

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

	plan := deployplan.NewPlan()
	populatePlan(ctx, plan, tfPlan.ResourceChanges)

	return plan, nil
}
