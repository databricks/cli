package terraform

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

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
	{"postgres_project_", "postgres_projects"},
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

// convertSecretAclNameToScopeKey converts terraform secret ACL resource names to scope permission keys.
// ACL names have format "secret_acl_<scope_key>_<idx>" (see convert_secret_scope.go).
// e.g., "secret_acl_my_scope_0" -> "resources.secret_scopes.my_scope.permissions"
func convertSecretAclNameToScopeKey(name string) string {
	name, _ = strings.CutPrefix(name, "secret_acl_")
	if i := strings.LastIndex(name, "_"); i >= 0 {
		name = name[:i]
	}
	return "resources.secret_scopes." + name + ".permissions"
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
			actionType = deployplan.Delete
		case rc.Change.Actions.Replace():
			actionType = deployplan.Recreate
		case rc.Change.Actions.Create():
			actionType = deployplan.Create
		case rc.Change.Actions.Update():
			actionType = deployplan.Update
		case rc.Change.Actions.NoOp():
			actionType = deployplan.Skip
		default:
			continue
		}

		group, ok := TerraformToGroupName[rc.Type]
		if !ok {
			log.Warnf(ctx, "unknown resource type '%s'", rc.Type)
			continue
		}

		var key string
		switch group {
		case "permissions":
			key = convertPermissionsResourceNameToKey(rc.Name)
		case "grants":
			key = convertGrantsResourceNameToKey(rc.Name)
		case "secret_acls":
			key = convertSecretAclNameToScopeKey(rc.Name)
		default:
			key = "resources." + group + "." + rc.Name
		}

		if existing, ok := plan.Plan[key]; ok {
			// For secret ACLs, multiple individual ACL changes are merged into a single
			// scope-level permissions entry. When the actions differ (e.g., some ACLs are
			// recreated while others are deleted), it means permissions are being updated,
			// not deleted entirely.
			if group == "secret_acls" && existing.Action != actionType {
				existing.Action = deployplan.Update
			} else {
				existing.Action = deployplan.GetHigherAction(existing.Action, actionType)
			}
		} else {
			plan.Plan[key] = &deployplan.PlanEntry{Action: actionType}
		}
	}
}

// ShowPlanFile reads a Terraform plan file located at planPath using the provided tfexec.Terraform handle
// and converts it into a deployplan.Plan.
func ShowPlanFile(ctx context.Context, tf *tfexec.Terraform, planPath string) (*deployplan.Plan, error) {
	tfPlan, err := tf.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, err
	}

	plan := deployplan.NewPlanTerraform()
	populatePlan(ctx, plan, tfPlan.ResourceChanges)

	return plan, nil
}
