package terraform

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestResourceChangeEventForCreateJob(t *testing.T) {
	tfChange := tfjson.ResourceChange{
		Change: &tfjson.Change{
			Actions: []tfjson.Action{"create"},
		},
		Type: "databricks_job",
		Name: "foo",
	}
	bundleChange := toResourceChangeEvent(&tfChange)

	assert.Equal(t, "create", string(bundleChange.Action))
	assert.Equal(t, "job", string(bundleChange.ResourceType))
	assert.Equal(t, "foo", string(bundleChange.Name))
}

func TestResourceChangeEventForReplacePipeline(t *testing.T) {
	tfChange := tfjson.ResourceChange{
		Change: &tfjson.Change{
			Actions: []tfjson.Action{"delete", "create"},
		},
		Type: "databricks_pipeline",
		Name: "bar",
	}
	bundleChange := toResourceChangeEvent(&tfChange)

	assert.Equal(t, "replace", string(bundleChange.Action))
	assert.Equal(t, "pipeline", string(bundleChange.ResourceType))
	assert.Equal(t, "bar", string(bundleChange.Name))
}

func TesResourceChangeEventForDeleteExperiment(t *testing.T) {
	tfChange := tfjson.ResourceChange{
		Change: &tfjson.Change{
			Actions: []tfjson.Action{"delete"},
		},
		Type: "databricks_mlflow_experiment",
		Name: "my_exp",
	}
	bundleChange := toResourceChangeEvent(&tfChange)

	assert.Equal(t, "delete", string(bundleChange.Action))
	assert.Equal(t, "mlflow_experiment", string(bundleChange.ResourceType))
	assert.Equal(t, "my_exp", string(bundleChange.Name))
}

func TestResourceChangeEventToString(t *testing.T) {
	event := ResourceChangeEvent{
		Name:         "foo",
		ResourceType: "job",
		Action:       ActionCreate,
	}
	assert.Equal(t, "  create job foo", event.String())
}
