package terraform

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestPopulatePlan(t *testing.T) {
	ctx := context.Background()
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

	plan := &deployplan.Plan{
		Plan: make(map[string]deployplan.PlanEntry),
	}

	populatePlan(ctx, plan, changes)

	actions := plan.GetActions()
	res := deployplan.FilterGroup(actions, "pipelines", deployplan.ActionTypeDelete, deployplan.ActionTypeRecreate)

	assert.Equal(t, []deployplan.Action{
		{
			ActionType: deployplan.ActionTypeDelete,
			ResourceNode: deployplan.ResourceNode{
				Group: "pipelines",
				Key:   "delete pipeline",
			},
		},
		{
			ActionType: deployplan.ActionTypeRecreate,
			ResourceNode: deployplan.ResourceNode{
				Group: "pipelines",
				Key:   "recreate pipeline",
			},
		},
	}, res)

	// Also test that the plan was populated correctly with expected entries
	assert.Contains(t, plan.Plan, "resources.pipelines.create pipeline")
	assert.Equal(t, "create", plan.Plan["resources.pipelines.create pipeline"].Action)

	assert.Contains(t, plan.Plan, "resources.pipelines.delete pipeline")
	assert.Equal(t, "delete", plan.Plan["resources.pipelines.delete pipeline"].Action)

	assert.Contains(t, plan.Plan, "resources.pipelines.recreate pipeline")
	assert.Equal(t, "recreate", plan.Plan["resources.pipelines.recreate pipeline"].Action)

	// Unknown resource type should not be in the plan
	assert.NotContains(t, plan.Plan, "resources.recreate whatever")
}
