package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
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
						ID:               "4",
						CreateExperiment: ml.CreateExperiment{Name: "experiment1"},
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
				VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
					"vectorsearchindex1": {
						ID: "catalog.schema.vectorsearchindex1",
						CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
							Name: "catalog.schema.vectorsearchindex1",
						},
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
							DisplayName: "My special dashboard",
						},
					},
				},
			},
		},
	}

	expectedURLs := map[string]string{
		"job1":               "https://mycompany.databricks.com/jobs/1?w=123456",
		"pipeline1":          "https://mycompany.databricks.com/pipelines/3?w=123456",
		"experiment1":        "https://mycompany.databricks.com/ml/experiments/4?w=123456",
		"model1":             "https://mycompany.databricks.com/ml/models/a%20model%20uses%20its%20name%20for%20identifier?w=123456",
		"servingendpoint1":   "https://mycompany.databricks.com/ml/endpoints/my_serving_endpoint?w=123456",
		"registeredmodel1":   "https://mycompany.databricks.com/explore/data/models/8?w=123456",
		"qualityMonitor1":    "https://mycompany.databricks.com/explore/data/catalog/schema/qualityMonitor1?w=123456",
		"vectorsearchindex1": "https://mycompany.databricks.com/explore/data/catalog/schema/vectorsearchindex1?w=123456",
		"schema1":            "https://mycompany.databricks.com/explore/data/catalog/schema?w=123456",
		"cluster1":           "https://mycompany.databricks.com/compute/clusters/1017-103929-vlr7jzcf?w=123456",
		"dashboard1":         "https://mycompany.databricks.com/dashboardsv3/01ef8d56871e1d50ae30ce7375e42478/published?w=123456",
	}

	err := initializeForWorkspace(b, "123456", "https://mycompany.databricks.com/")
	require.NoError(t, err)

	for _, group := range b.Config.Resources.AllResources() {
		for key, r := range group.Resources {
			url, _ := r.GetURL()
			require.Equal(t, expectedURLs[key], url, "Unexpected URL for "+key)
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

// TestInitializeURLsApplyNonNumericConfigPassedThrough verifies that a
// non-numeric Config.WorkspaceID (e.g. a UUID connection-style identifier) is
// passed through unchanged into the ?w= parameter. The numeric-mismatch check
// is skipped because such IDs cannot be compared against an integer org ID.
func TestInitializeURLsApplyNonNumericConfigPassedThrough(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: "some-uuid-style-id",
	})
	require.NoError(t, err)

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
	b.SetWorkpaceClient(w)

	diags := InitializeURLs().Apply(t.Context(), b)
	require.NoError(t, diags.Error())

	// UUID flows through into ?w= unchanged — no numeric comparison is possible.
	require.Equal(t,
		server.URL+"/jobs/1?w=some-uuid-style-id",
		b.Config.Resources.Jobs["job1"].URL,
	)
}

// TestInitializeURLsApplyNoneSentinelResolvesViaAPI verifies that the "none"
// sentinel (persisted to .databrickscfg when a user explicitly skips
// workspace selection) is treated as unset rather than as a literal value: it
// must not end up as ?w=none, and it must not be compared against the API
// result (both would be wrong; the sentinel means "no preference").
func TestInitializeURLsApplyNoneSentinelResolvesViaAPI(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	// /Me returns X-Databricks-Org-Id: 900800700600.

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: "none",
	})
	require.NoError(t, err)

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
	b.SetWorkpaceClient(w)

	diags := InitializeURLs().Apply(t.Context(), b)
	require.NoError(t, diags.Error())

	require.Equal(t,
		server.URL+"/jobs/1?w=900800700600",
		b.Config.Resources.Jobs["job1"].URL,
	)
}

// TestInitializeURLsApplyErrorsOnNumericWorkspaceIDMismatch verifies that Apply
// returns an error when Config.WorkspaceID is a numeric value that differs from
// the workspace org ID returned by the API. This prevents silently embedding a
// wrong ?w= parameter in resource URLs that would navigate to an unexpected workspace.
func TestInitializeURLsApplyErrorsOnNumericWorkspaceIDMismatch(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	// /Me returns X-Databricks-Org-Id: 900800700600.

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: "12345", // numeric but does not match 900800700600
	})
	require.NoError(t, err)

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
	b.SetWorkpaceClient(w)

	diags := InitializeURLs().Apply(t.Context(), b)
	require.ErrorContains(t, diags.Error(), "12345")
	require.ErrorContains(t, diags.Error(), "900800700600")
	require.ErrorContains(t, diags.Error(), "disambiguate")
}
