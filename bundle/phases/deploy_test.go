package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestParseTerraformActions(t *testing.T) {
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

	actions := terraform.GetActions(t.Context(), changes)
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
}
