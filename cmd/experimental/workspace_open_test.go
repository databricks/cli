package experimental

import (
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
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
		{"jobs", "123", "https://myworkspace.databricks.com/jobs/123"},
		{"pipelines", "abc-def", "https://myworkspace.databricks.com/pipelines/abc-def"},
		{"dashboards", "dash-1", "https://myworkspace.databricks.com/dashboardsv3/dash-1/published"},
		{"experiments", "exp-1", "https://myworkspace.databricks.com/ml/experiments/exp-1"},
		{"warehouses", "wh-1", "https://myworkspace.databricks.com/sql/warehouses/wh-1"},
		{"queries", "q-1", "https://myworkspace.databricks.com/sql/editor/q-1"},
		{"apps", "my-app", "https://myworkspace.databricks.com/apps/my-app"},
		{"clusters", "0123-456789-abc", "https://myworkspace.databricks.com/compute/clusters/0123-456789-abc"},
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
		{"notebooks", "12345", "https://myworkspace.databricks.com/#notebook/12345"},
		{"notebooks", "/Users/user@example.com/my-notebook", "https://myworkspace.databricks.com/#notebook//Users/user@example.com/my-notebook"},
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
	assert.ErrorContains(t, err, "apps, clusters, dashboards, experiments, jobs, notebooks, pipelines, queries, warehouses")
}

func TestBuildWorkspaceURLHostWithTrailingSlash(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com/", "jobs", "123", 0)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceID(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com", "jobs", "123", 123456)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123?o=123456", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInHostname(t *testing.T) {
	got, err := buildWorkspaceURL("https://adb-123456.azuredatabricks.net", "jobs", "123", 123456)
	require.NoError(t, err)
	// Workspace ID is already in the hostname, so ?o= should not be appended.
	assert.Equal(t, "https://adb-123456.azuredatabricks.net/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInVanityHostname(t *testing.T) {
	got, err := buildWorkspaceURL("https://workspace-123456.example.com", "jobs", "123", 123456)
	require.NoError(t, err)
	assert.Equal(t, "https://workspace-123456.example.com/jobs/123?o=123456", got)
}

func TestBuildWorkspaceURLFragmentWithWorkspaceID(t *testing.T) {
	got, err := buildWorkspaceURL("https://myworkspace.databricks.com", "notebooks", "12345", 789)
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/?o=789#notebook/12345", got)
}

func TestWorkspaceOpenCommandCompletion(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, completions, "jobs")
	assert.Contains(t, completions, "notebooks")
	assert.Contains(t, completions, "clusters")
	assert.Contains(t, completions, "pipelines")
	assert.Contains(t, completions, "dashboards")
	assert.Contains(t, completions, "experiments")
	assert.Contains(t, completions, "warehouses")
	assert.Contains(t, completions, "queries")
	assert.Contains(t, completions, "apps")
	assert.Len(t, completions, 9)
}

func TestWorkspaceOpenCommandCompletionSecondArg(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{"jobs"}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Nil(t, completions)
}

func TestWorkspaceOpenCommandHelpText(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	assert.Contains(t, cmd.Long, "Supported resource types: apps, clusters, dashboards, experiments, jobs, notebooks, pipelines, queries, warehouses.")
	assert.Contains(t, cmd.Long, "databricks experimental open jobs 123456789")
	assert.Contains(t, cmd.Long, "databricks experimental open notebooks /Users/user@example.com/my-notebook")
	assert.Contains(t, cmd.Long, "databricks experimental open jobs 123456789 --url")

	flag := cmd.Flags().Lookup("url")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestWorkspaceOpenCommandOpensBrowserByDefault(t *testing.T) {
	originalCurrentWorkspaceID := currentWorkspaceID
	originalOpenWorkspaceURL := openWorkspaceURL
	t.Cleanup(func() {
		currentWorkspaceID = originalCurrentWorkspaceID
		openWorkspaceURL = originalOpenWorkspaceURL
	})

	currentWorkspaceID = func(context.Context) (int64, error) {
		return 0, nil
	}

	var gotURL string
	openWorkspaceURL = func(ctx context.Context, targetURL string) error {
		gotURL = targetURL
		return nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &config.Config{
			Host: "https://myworkspace.databricks.com",
		},
	})

	cmd := newWorkspaceOpenCommand()
	cmd.SetContext(ctx)

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{"jobs", "123"})
	require.NoError(t, err)

	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123", gotURL)
	assert.Equal(t, "", stdout.String())
	assert.Contains(t, stderr.String(), "Opening jobs 123 in the browser...")
}

func TestWorkspaceOpenCommandURLFlag(t *testing.T) {
	originalCurrentWorkspaceID := currentWorkspaceID
	originalOpenWorkspaceURL := openWorkspaceURL
	t.Cleanup(func() {
		currentWorkspaceID = originalCurrentWorkspaceID
		openWorkspaceURL = originalOpenWorkspaceURL
	})

	currentWorkspaceID = func(context.Context) (int64, error) {
		return 789, nil
	}

	browserOpened := false
	openWorkspaceURL = func(ctx context.Context, targetURL string) error {
		browserOpened = true
		return nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &config.Config{
			Host: "https://myworkspace.databricks.com",
		},
	})

	cmd := newWorkspaceOpenCommand()
	cmd.SetContext(ctx)

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	require.NoError(t, cmd.Flags().Set("url", "true"))

	err := cmd.RunE(cmd, []string{"jobs", "123"})
	require.NoError(t, err)

	assert.False(t, browserOpened)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123?o=789\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}
