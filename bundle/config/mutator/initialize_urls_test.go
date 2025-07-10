package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
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
						JobSettings: jobs.JobSettings{Name: "job1"},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						ID:             "3",
						CreatePipeline: pipelines.CreatePipeline{Name: "pipeline1"},
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {
						ID:         "4",
						Experiment: ml.Experiment{Name: "experiment1"},
					},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {
						ID:                 "a model uses its name for identifier",
						CreateModelRequest: ml.CreateModelRequest{Name: "a model uses its name for identifier"},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"servingendpoint1": {
						ID: "my_serving_endpoint",
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "my_serving_endpoint",
						},
					},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"registeredmodel1": {
						ID: "8",
						CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
							Name: "my_registered_model",
						},
					},
				},
				QualityMonitors: map[string]*resources.QualityMonitor{
					"qualityMonitor1": {
						TableName:     "catalog.schema.qualityMonitor1",
						CreateMonitor: catalog.CreateMonitor{},
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {
						ID: "catalog.schema",
						CreateSchema: catalog.CreateSchema{
							Name: "schema",
						},
					},
				},
				Clusters: map[string]*resources.Cluster{
					"cluster1": {
						ID: "1017-103929-vlr7jzcf",
						ClusterSpec: compute.ClusterSpec{
							ClusterName: "cluster1",
						},
					},
				},
				Dashboards: map[string]*resources.Dashboard{
					"dashboard1": {
						ID: "01ef8d56871e1d50ae30ce7375e42478",
						DashboardConfig: resources.DashboardConfig{
							Dashboard: dashboards.Dashboard{
								DisplayName: "My special dashboard",
							},
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
		"model1":           "https://mycompany.databricks.com/ml/models/a%20model%20uses%20its%20name%20for%20identifier?o=123456",
		"servingendpoint1": "https://mycompany.databricks.com/ml/endpoints/my_serving_endpoint?o=123456",
		"registeredmodel1": "https://mycompany.databricks.com/explore/data/models/8?o=123456",
		"qualityMonitor1":  "https://mycompany.databricks.com/explore/data/catalog/schema/qualityMonitor1?o=123456",
		"schema1":          "https://mycompany.databricks.com/explore/data/catalog/schema?o=123456",
		"cluster1":         "https://mycompany.databricks.com/compute/clusters/1017-103929-vlr7jzcf?o=123456",
		"dashboard1":       "https://mycompany.databricks.com/dashboardsv3/01ef8d56871e1d50ae30ce7375e42478/published?o=123456",
	}

	err := initializeForWorkspace(b, "123456", "https://mycompany.databricks.com/")
	require.NoError(t, err)

	for _, group := range b.Config.Resources.AllResources() {
		for key, r := range group.Resources {
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
						JobSettings: jobs.JobSettings{Name: "job1"},
					},
				},
			},
		},
	}

	err := initializeForWorkspace(b, "123456", "https://adb-123456.azuredatabricks.net/")
	require.NoError(t, err)

	require.Equal(t, "https://adb-123456.azuredatabricks.net/jobs/1", b.Config.Resources.Jobs["job1"].URL)
}
