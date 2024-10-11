package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/require"
)

func TestInitializeURLs(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com/",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						ID:          "1",
						JobSettings: &jobs.JobSettings{Name: "job1"},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						ID:           "3",
						PipelineSpec: &pipelines.PipelineSpec{Name: "pipeline1"},
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {
						ID:         "4",
						Experiment: &ml.Experiment{Name: "experiment1"},
					},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {
						ID:    "6",
						Model: &ml.Model{Name: "model1"},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"servingendpoint1": {
						ID: "7",
						CreateServingEndpoint: &serving.CreateServingEndpoint{
							Name: "my_serving_endpoint",
						},
					},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"registeredmodel1": {
						ID: "8",
						CreateRegisteredModelRequest: &catalog.CreateRegisteredModelRequest{
							Name: "my_registered_model",
						},
					},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"qualityMonitor1": {
						CreateMonitor: &catalog.CreateMonitor{
							TableName: "catalog.schema.qualityMonitor1",
						},
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {
						ID: "catalog.schema",
						CreateSchema: &catalog.CreateSchema{
							Name: "schema",
						},
					},
				},
			},
		},
	}

	expectedURLs := map[string]string{
		"job1":             "https://mycompany.databricks.com/jobs/1?o=123456",
		"pipeline1":        "https://mycompany.databricks.com/pipelines/3?o=123456",
		"experiment1":      "https://mycompany.databricks.com/ml/experiments/4?o=123456",
		"model1":           "https://mycompany.databricks.com/ml/models/model1?o=123456",
		"servingendpoint1": "https://mycompany.databricks.com/ml/endpoints/my_serving_endpoint?o=123456",
		"registeredmodel1": "https://mycompany.databricks.com/explore/data/models/8?o=123456",
		"qualityMonitor1":  "https://mycompany.databricks.com/explore/data/catalog/schema/qualityMonitor1?o=123456",
		"schema1":          "https://mycompany.databricks.com/explore/data/catalog/schema?o=123456",
	}

	initializeForWorkspace(b, "123456", "https://mycompany.databricks.com/")

	for _, rs := range b.Config.Resources.AllResources() {
		for key, r := range rs {
			require.Equal(t, expectedURLs[key], r.GetURL(), "Unexpected URL for "+key)
		}
	}
}

func TestInitializeURLsWithoutOrgId(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						ID:          "1",
						JobSettings: &jobs.JobSettings{Name: "job1"},
					},
				},
			},
		},
	}

	initializeForWorkspace(b, "123456", "https://adb-123456.azuredatabricks.net/")

	require.Equal(t, "https://adb-123456.azuredatabricks.net/jobs/1", b.Config.Resources.Jobs["job1"].URL)
}
