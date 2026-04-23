package terraform

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm/deployplan"
	tfjson "github.com/hashicorp/terraform-json"
)

// terraformToGroupName maps a databricks_* terraform resource type back to the
// ucm resource group key used under `resources.<group>.<name>`. ucm currently
// emits only these three; extend here when new tfdyn converters land.
//
// Forked from bundle/deploy/terraform's TerraformToGroupName map rather than
// imported — the DAB map covers jobs/pipelines/etc but not databricks_catalog,
// and pinning to a bundle map will drift as upstream adds resources ucm does
// not model. See cmd/ucm/CLAUDE.md on fork isolation.
var terraformToGroupName = map[string]string{
	"databricks_catalog":            "catalogs",
	"databricks_schema":             "schemas",
	"databricks_grants":             "grants",
	"databricks_storage_credential": "storage_credentials",
	"databricks_external_location":  "external_locations",
	"databricks_volume":             "volumes",
	"databricks_connection":         "connections",
}

// populatePlan fills `plan` from terraform resource changes using the
// ucm resource-key convention. Unknown resource types are logged and skipped
// so a stray TF resource (e.g. a provider added a new type) doesn't hard-fail
// the command.
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

		group, ok := terraformToGroupName[rc.Type]
		if !ok {
			log.Warnf(ctx, "unknown resource type %q", rc.Type)
			continue
		}

		key := "resources." + group + "." + rc.Name

		if existing, ok := plan.Plan[key]; ok {
			existing.Action = deployplan.GetHigherAction(existing.Action, actionType)
		} else {
			plan.Plan[key] = &deployplan.PlanEntry{Action: actionType}
		}
	}
}

// translatePlan turns a parsed terraform plan into a ucm deployplan.Plan.
// Mirrors bundle/deploy/terraform.ShowPlanFile's tail half; the show-plan
// read itself lives on the runner so tests can stub it.
func translatePlan(ctx context.Context, tfPlan *tfjson.Plan) *deployplan.Plan {
	plan := deployplan.NewPlanTerraform()
	populatePlan(ctx, plan, tfPlan.ResourceChanges)
	return plan
}
