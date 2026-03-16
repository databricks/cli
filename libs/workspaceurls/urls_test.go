package workspaceurls

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDotSeparatedID(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		id           string
		expected     string
	}{
		{"registered_models converts dots to slashes", ResourceRegisteredModels, "catalog.schema.model", "catalog/schema/model"},
		{"registered_models preserves slashes", ResourceRegisteredModels, "catalog/schema/model", "catalog/schema/model"},
		{"registered_models single part", ResourceRegisteredModels, "model", "model"},
		{"jobs ID unchanged", ResourceJobs, "123", "123"},
		{"pipelines ID unchanged", ResourcePipelines, "abc-def", "abc-def"},
		{"notebooks path unchanged", ResourceNotebooks, "/Users/user@example.com/nb", "/Users/user@example.com/nb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeDotSeparatedID(tt.resourceType, tt.id)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLookupPattern(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     string
		ok           bool
	}{
		{ResourceAlerts, AlertPattern, true},
		{ResourceApps, AppPattern, true},
		{ResourceClusters, ClusterPattern, true},
		{ResourceDashboards, DashboardPattern, true},
		{ResourceExperiments, ExperimentPattern, true},
		{ResourceJobs, JobPattern, true},
		{ResourceModels, ModelPattern, true},
		{ResourceModelServingEndpoints, ModelServingEndpointPattern, true},
		{ResourceNotebooks, NotebookPattern, true},
		{ResourcePipelines, PipelinePattern, true},
		{ResourceQueries, QueryPattern, true},
		{ResourceRegisteredModels, RegisteredModelPattern, true},
		{ResourceWarehouses, WarehousePattern, true},
		{"unknown", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			pattern, ok := LookupPattern(tt.resourceType)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, pattern)
		})
	}
}

func TestSortResourceTypes(t *testing.T) {
	input := []string{"jobs", "alerts", "clusters"}
	got := SortResourceTypes(input)
	assert.Equal(t, []string{"alerts", "clusters", "jobs"}, got)

	// Original slice is not modified.
	assert.Equal(t, []string{"jobs", "alerts", "clusters"}, input)
}

func TestSortResourceTypesEmpty(t *testing.T) {
	got := SortResourceTypes(nil)
	assert.Empty(t, got)
}

func TestWorkspaceBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		workspaceID int64
		expected    string
	}{
		{"no workspace ID", "https://myworkspace.databricks.com", 0, "https://myworkspace.databricks.com"},
		{"with workspace ID", "https://myworkspace.databricks.com", 123456, "https://myworkspace.databricks.com?o=123456"},
		{"trailing slash stripped", "https://myworkspace.databricks.com/", 0, "https://myworkspace.databricks.com/"},
		{"trailing slash with workspace ID", "https://myworkspace.databricks.com/", 789, "https://myworkspace.databricks.com/?o=789"},
		{"adb hostname skips query param", "https://adb-123456.azuredatabricks.net", 123456, "https://adb-123456.azuredatabricks.net"},
		{"adb hostname mismatch adds param", "https://adb-999.azuredatabricks.net", 123456, "https://adb-999.azuredatabricks.net?o=123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WorkspaceBaseURL(tt.host, tt.workspaceID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got.String())
		})
	}
}

func TestWorkspaceBaseURLInvalidHost(t *testing.T) {
	_, err := WorkspaceBaseURL("://invalid", 0)
	assert.ErrorContains(t, err, "invalid workspace host")
}

func TestBuildResourceURL(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		pattern     string
		id          string
		workspaceID int64
		expected    string
	}{
		{"simple path", "https://host.com", "jobs/%s", "123", 0, "https://host.com/jobs/123"},
		{"path with workspace ID", "https://host.com", "jobs/%s", "123", 456, "https://host.com/jobs/123?o=456"},
		{"fragment pattern", "https://host.com", "#notebook/%s", "12345", 0, "https://host.com/#notebook/12345"},
		{"fragment with workspace ID", "https://host.com", "#notebook/%s", "12345", 789, "https://host.com/?o=789#notebook/12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildResourceURL(tt.host, tt.pattern, tt.id, tt.workspaceID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResourceURL(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		id       string
		expected string
	}{
		{"simple path", "jobs/%s", "123", "https://host.com/jobs/123"},
		{"nested path", "ml/experiments/%s", "exp-1", "https://host.com/ml/experiments/exp-1"},
		{"published dashboard", "dashboardsv3/%s/published", "d-1", "https://host.com/dashboardsv3/d-1/published"},
		{"fragment", "#notebook/%s", "12345", "https://host.com/#notebook/12345"},
		{"fragment with path ID", "#notebook/%s", "/Users/u/nb", "https://host.com/#notebook//Users/u/nb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := url.URL{Scheme: "https", Host: "host.com"}
			got := ResourceURL(base, tt.pattern, tt.id)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestHasWorkspaceIDInHostname(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		workspaceID string
		expected    bool
	}{
		{"matching adb prefix", "adb-123456.azuredatabricks.net", "123456", true},
		{"matching adb uppercase", "ADB-123456.azuredatabricks.net", "123456", true},
		{"different workspace ID", "adb-999.azuredatabricks.net", "123456", false},
		{"no adb prefix", "myworkspace.databricks.com", "123456", false},
		{"partial match in subdomain", "adb-123456789.azuredatabricks.net", "123456", false},
		{"adb prefix only hostname", "adb-123456", "123456", true},
		{"empty hostname", "", "123456", false},
		{"empty workspace ID", "adb-.azuredatabricks.net", "", true},
		{"vanity hostname with ID", "workspace-123456.example.com", "123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasWorkspaceIDInHostname(tt.hostname, tt.workspaceID)
			assert.Equal(t, tt.expected, got)
		})
	}
}
