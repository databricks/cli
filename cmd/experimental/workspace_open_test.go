package experimental

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWorkspaceURLPathBasedResources(t *testing.T) {
	tests := []struct {
		resourceType string
		id           string
		expected     string
	}{
		{"job", "123", "https://myworkspace.databricks.com/jobs/123"},
		{"pipeline", "abc-def", "https://myworkspace.databricks.com/pipelines/abc-def"},
		{"dashboard", "dash-1", "https://myworkspace.databricks.com/dashboardsv3/dash-1/published"},
		{"warehouse", "wh-1", "https://myworkspace.databricks.com/sql/warehouses/wh-1"},
		{"query", "q-1", "https://myworkspace.databricks.com/sql/editor/q-1"},
		{"app", "my-app", "https://myworkspace.databricks.com/apps/my-app"},
		{"cluster", "0123-456789-abc", "https://myworkspace.databricks.com/compute/clusters/0123-456789-abc"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got, err := buildWorkspaceURL("https://myworkspace.databricks.com", tt.resourceType, tt.id, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildWorkspaceURLFragmentBasedResources(t *testing.T) {
	tests := []struct {
		resourceType string
		id           string
		expected     string
	}{
		{"notebook", "12345", "https://myworkspace.databricks.com/#notebook/12345"},
		{"notebook", "/Users/user@example.com/my-notebook", "https://myworkspace.databricks.com/#notebook//Users/user@example.com/my-notebook"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, err := buildWorkspaceURL("https://myworkspace.databricks.com", tt.resourceType, tt.id, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildWorkspaceURLUnknownResourceType(t *testing.T) {
	_, err := buildWorkspaceURL("https://myworkspace.databricks.com", "unknown", "123", 0)
	assert.ErrorContains(t, err, "unknown resource type \"unknown\"")
}

func TestBuildWorkspaceURLHostWithTrailingSlash(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com/", "job", "123", 0)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceID(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com", "job", "123", 123456)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123?o=123456", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInHostname(t *testing.T) {
	got, err := buildWorkspaceURL("https://adb-123456.azuredatabricks.net", "job", "123", 123456)
	require.NoError(t, err)
	// Workspace ID is already in the hostname, so ?o= should not be appended.
	assert.Equal(t, "https://adb-123456.azuredatabricks.net/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInVanityHostname(t *testing.T) {
	got, err := buildWorkspaceURL("https://workspace-123456.example.com", "job", "123", 123456)
	require.NoError(t, err)
	assert.Equal(t, "https://workspace-123456.example.com/jobs/123?o=123456", got)
}

func TestBuildWorkspaceURLFragmentWithWorkspaceID(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com", "notebook", "12345", 789)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/?o=789#notebook/12345", got)
}

func TestWorkspaceOpenCommandCompletion(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, completions, "job")
	assert.Contains(t, completions, "notebook")
	assert.Contains(t, completions, "cluster")
	assert.Contains(t, completions, "pipeline")
	assert.Contains(t, completions, "dashboard")
	assert.Contains(t, completions, "warehouse")
	assert.Contains(t, completions, "query")
	assert.Contains(t, completions, "app")
	assert.Len(t, completions, 8)
}

func TestWorkspaceOpenCommandCompletionSecondArg(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{"job"}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Nil(t, completions)
}
