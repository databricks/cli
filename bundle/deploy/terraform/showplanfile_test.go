package terraform

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestPopulatePlan(t *testing.T) {
	ctx := t.Context()
	changes := []*tfjson.ResourceChange{
		{
			Type: "databricks_pipeline",
			Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionCreate},
			},
			Name: "create pipeline",
		},
		{
			Type: "databricks_pipeline",
			Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionDelete},
			},
			Name: "delete pipeline",
		},
		{
			Type: "databricks_pipeline",
			Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate},
			},
			Name: "recreate pipeline",
		},
		{
			Type: "databricks_whatever",
			Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate},
			},
			Name: "recreate whatever",
		},
	}

	plan := deployplan.NewPlanTerraform()
	populatePlan(ctx, plan, changes)

	actions := plan.GetActions()

	// Assert that the actions list contains all expected actions
	expectedActions := []deployplan.Action{
		{
			ActionType:  deployplan.Create,
			ResourceKey: "resources.pipelines.create pipeline",
		},
		{
			ActionType:  deployplan.Delete,
			ResourceKey: "resources.pipelines.delete pipeline",
		},
		{
			ActionType:  deployplan.Recreate,
			ResourceKey: "resources.pipelines.recreate pipeline",
		},
	}
	assert.Equal(t, expectedActions, actions)

	// Also test that the plan was populated correctly with expected entries
	assert.Contains(t, plan.Plan, "resources.pipelines.create pipeline")
	assert.Equal(t, deployplan.Create, plan.Plan["resources.pipelines.create pipeline"].Action)

	assert.Contains(t, plan.Plan, "resources.pipelines.delete pipeline")
	assert.Equal(t, deployplan.Delete, plan.Plan["resources.pipelines.delete pipeline"].Action)

	assert.Contains(t, plan.Plan, "resources.pipelines.recreate pipeline")
	assert.Equal(t, deployplan.Recreate, plan.Plan["resources.pipelines.recreate pipeline"].Action)

	// Unknown resource type should not be in the plan
	assert.NotContains(t, plan.Plan, "resources.recreate whatever")
}

func TestPopulatePlanSecretAcl(t *testing.T) {
	ctx := t.Context()
	changes := []*tfjson.ResourceChange{
		{
			Type:   "databricks_secret_acl",
			Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
			Name:   "secret_acl_my_scope_0",
		},
		{
			Type:   "databricks_secret_acl",
			Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}},
			Name:   "secret_acl_my_scope_1",
		},
	}

	plan := deployplan.NewPlanTerraform()
	populatePlan(ctx, plan, changes)

	// Multiple ACL changes for the same scope are merged with highest severity.
	assert.Equal(t, map[string]*deployplan.PlanEntry{
		"resources.secret_scopes.my_scope.permissions": {Action: deployplan.Recreate},
	}, plan.Plan)
}

func TestPopulatePlanSecretAclMixedCreateDelete(t *testing.T) {
	ctx := t.Context()
	changes := []*tfjson.ResourceChange{
		{
			Type:   "databricks_secret_acl",
			Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}},
			Name:   "secret_acl_my_scope_0",
		},
		{
			Type:   "databricks_secret_acl",
			Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
			Name:   "secret_acl_my_scope_1",
		},
	}

	plan := deployplan.NewPlanTerraform()
	populatePlan(ctx, plan, changes)

	assert.Equal(t, map[string]*deployplan.PlanEntry{
		"resources.secret_scopes.my_scope.permissions": {Action: deployplan.Update},
	}, plan.Plan)
}

func TestConvertSecretAclNameToScopeKey(t *testing.T) {
	assert.Equal(t, "resources.secret_scopes.my_scope.permissions", convertSecretAclNameToScopeKey("secret_acl_my_scope_0"))
	assert.Equal(t, "resources.secret_scopes.my_scope.permissions", convertSecretAclNameToScopeKey("secret_acl_my_scope_1"))
	assert.Equal(t, "resources.secret_scopes.scope_123.permissions", convertSecretAclNameToScopeKey("secret_acl_scope_123_2"))
}
