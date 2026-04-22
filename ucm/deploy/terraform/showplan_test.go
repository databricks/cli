package terraform

import (
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deployplan"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func change(actions tfjson.Actions) *tfjson.Change {
	return &tfjson.Change{Actions: actions}
}

func TestPopulatePlan_MapsActionsAndKeys(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	plan := deployplan.NewPlanTerraform()

	populatePlan(ctx, plan, []*tfjson.ResourceChange{
		{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionCreate})},
		{Type: "databricks_schema", Name: "sales_raw", Change: change(tfjson.Actions{tfjson.ActionUpdate})},
		{Type: "databricks_grants", Name: "analysts", Change: change(tfjson.Actions{tfjson.ActionDelete})},
		{Type: "databricks_storage_credential", Name: "sales_cred", Change: change(tfjson.Actions{tfjson.ActionCreate})},
	})

	assert.Equal(t, deployplan.Create, plan.Plan["resources.catalogs.main"].Action)
	assert.Equal(t, deployplan.Update, plan.Plan["resources.schemas.sales_raw"].Action)
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.grants.analysts"].Action)
	assert.Equal(t, deployplan.Create, plan.Plan["resources.storage_credentials.sales_cred"].Action)
}

func TestPopulatePlan_ReplaceBecomesRecreate(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	plan := deployplan.NewPlanTerraform()

	populatePlan(ctx, plan, []*tfjson.ResourceChange{
		{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate})},
	})

	assert.Equal(t, deployplan.Recreate, plan.Plan["resources.catalogs.main"].Action)
}

func TestPopulatePlan_NoOpBecomesSkip(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	plan := deployplan.NewPlanTerraform()

	populatePlan(ctx, plan, []*tfjson.ResourceChange{
		{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionNoop})},
	})

	assert.Equal(t, deployplan.Skip, plan.Plan["resources.catalogs.main"].Action)
}

// TestPopulatePlan_UnknownTypeIsSkipped guards the "new TF resource type from
// an upgraded provider" case — a warning is logged and the plan is unchanged
// rather than hard-failing the verb.
func TestPopulatePlan_UnknownTypeIsSkipped(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	plan := deployplan.NewPlanTerraform()

	populatePlan(ctx, plan, []*tfjson.ResourceChange{
		{Type: "databricks_nonexistent", Name: "x", Change: change(tfjson.Actions{tfjson.ActionCreate})},
	})

	assert.Empty(t, plan.Plan)
}

// TestPopulatePlan_MergesHigherSeverity mirrors bundle's GetHigherAction-based
// merging for the edge case where a single resource key appears twice with
// different actions (a defensive invariant — ucm tfdyn shouldn't emit dupes).
func TestPopulatePlan_MergesHigherSeverity(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	plan := deployplan.NewPlanTerraform()

	populatePlan(ctx, plan, []*tfjson.ResourceChange{
		{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionUpdate})},
		{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionDelete})},
	})

	assert.Equal(t, deployplan.Delete, plan.Plan["resources.catalogs.main"].Action)
}

func TestTranslatePlan_WrapsPopulate(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	tfPlan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{Type: "databricks_catalog", Name: "main", Change: change(tfjson.Actions{tfjson.ActionCreate})},
		},
	}

	plan := translatePlan(ctx, tfPlan)
	assert.Equal(t, deployplan.Create, plan.Plan["resources.catalogs.main"].Action)
}
