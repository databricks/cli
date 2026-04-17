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
		{"registered_models converts dots to slashes", "registered_models", "catalog.schema.model", "catalog/schema/model"},
		{"registered_models preserves slashes", "registered_models", "catalog/schema/model", "catalog/schema/model"},
		{"registered_models single part", "registered_models", "model", "model"},
		{"jobs ID unchanged", "jobs", "123", "123"},
		{"pipelines ID unchanged", "pipelines", "abc-def", "abc-def"},
		{"notebooks path unchanged", "notebooks", "/Users/user@example.com/nb", "/Users/user@example.com/nb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDotSeparatedID(tt.resourceType, tt.id)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResourceTypes(t *testing.T) {
	types := ResourceTypes()
	assert.NotEmpty(t, types)

	// Verify the list is sorted.
	for i := range len(types) - 1 {
		assert.Less(t, types[i], types[i+1])
	}
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
			got, err := workspaceBaseURL(tt.host, tt.workspaceID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got.String())
		})
	}
}

func TestWorkspaceBaseURLInvalidHost(t *testing.T) {
	_, err := workspaceBaseURL("://invalid", 0)
	assert.ErrorContains(t, err, "invalid workspace host")
}

func TestBuildResourceURL(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		resourceType string
		id           string
		workspaceID  int64
		expected     string
	}{
		{"simple path", "https://host.com", "jobs", "123", 0, "https://host.com/jobs/123"},
		{"path with workspace ID", "https://host.com", "jobs", "123", 456, "https://host.com/jobs/123?o=456"},
		{"fragment pattern", "https://host.com", "notebooks", "12345", 0, "https://host.com/#notebook/12345"},
		{"fragment with workspace ID", "https://host.com", "notebooks", "12345", 789, "https://host.com/?o=789#notebook/12345"},
		{"registered model normalizes dots", "https://host.com", "registered_models", "catalog.schema.model", 0, "https://host.com/explore/data/models/catalog/schema/model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildResourceURL(tt.host, tt.resourceType, tt.id, tt.workspaceID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildResourceURLUnknownType(t *testing.T) {
	_, err := BuildResourceURL("https://host.com", "unknown", "123", 0)
	assert.ErrorContains(t, err, "unknown resource type")
}

func TestResourceURL(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		id           string
		expected     string
	}{
		{"jobs", "jobs", "123", "https://host.com/jobs/123"},
		{"experiments", "experiments", "exp-1", "https://host.com/ml/experiments/exp-1"},
		{"dashboards", "dashboards", "d-1", "https://host.com/dashboardsv3/d-1/published"},
		{"notebooks", "notebooks", "12345", "https://host.com/#notebook/12345"},
		{"notebooks with path", "notebooks", "/Users/u/nb", "https://host.com/#notebook//Users/u/nb"},
		{"registered_models normalizes dots", "registered_models", "cat.sch.model", "https://host.com/explore/data/models/cat/sch/model"},
		{"sql_warehouses alias resolves to warehouses", "sql_warehouses", "wh-1", "https://host.com/sql/warehouses/wh-1"},
		{"warehouses canonical still works", "warehouses", "wh-1", "https://host.com/sql/warehouses/wh-1"},
		{"unknown returns empty", "nonexistent", "123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := url.URL{Scheme: "https", Host: "host.com"}
			got := ResourceURL(base, tt.resourceType, tt.id)
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
