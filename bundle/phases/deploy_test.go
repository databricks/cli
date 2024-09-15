package phases

import (
	"testing"

	terraformlib "github.com/databricks/cli/libs/terraform"
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

	res := filterDeleteOrRecreateActions(changes, "databricks_pipeline")

	assert.Equal(t, []terraformlib.Action{
		{
			Action:       terraformlib.ActionTypeDelete,
			ResourceType: "databricks_pipeline",
			ResourceName: "delete pipeline",
		},
		{
			Action:       terraformlib.ActionTypeRecreate,
			ResourceType: "databricks_pipeline",
			ResourceName: "recreate pipeline",
		},
	}, res)
}
