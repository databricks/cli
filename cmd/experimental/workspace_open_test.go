package experimental

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/workspaceurls"
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
		{"registered_models", "catalog.schema.model", "https://myworkspace.databricks.com/explore/data/models/catalog/schema/model"},
		{"features", "catalog.schema.feature", "https://myworkspace.databricks.com/explore/data/features/catalog/schema/feature"},
		{"feature_specs", "catalog.schema.spec", "https://myworkspace.databricks.com/explore/data/functions/catalog/schema/spec"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType+"/"+tt.id, func(t *testing.T) {
			got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", tt.resourceType, tt.id, "")
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
			got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", tt.resourceType, tt.id, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildWorkspaceURLUnknownResourceType(t *testing.T) {
	_, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", "unknown", "123", "")
	assert.ErrorContains(t, err, "unknown resource type \"unknown\"")
	assert.ErrorContains(t, err, strings.Join(workspaceOpenResourceTypes(), ", "))
}

func TestBuildWorkspaceURLHostWithTrailingSlash(t *testing.T) {
	got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com/", "jobs", "123", "")
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceID(t *testing.T) {
	got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", "jobs", "123", "123456")
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123?w=123456", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInHostname(t *testing.T) {
	got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://adb-123456.azuredatabricks.net", "jobs", "123", "123456")
	require.NoError(t, err)
	// Workspace ID is already in the hostname, so ?w= should not be appended.
	assert.Equal(t, "https://adb-123456.azuredatabricks.net/jobs/123", got)
}

func TestBuildWorkspaceURLWithWorkspaceIDInVanityHostname(t *testing.T) {
	got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://workspace-123456.example.com", "jobs", "123", "123456")
	require.NoError(t, err)
	assert.Equal(t, "https://workspace-123456.example.com/jobs/123?w=123456", got)
}

func TestBuildWorkspaceURLFragmentWithWorkspaceID(t *testing.T) {
	got, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", "notebooks", "12345", "789")
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/?w=789#notebook/12345", got)
}

// workspaceOpenResourceTypes returns the resource types the command accepts. The
// full list is pinned by the acceptance test of the command.
func workspaceOpenResourceTypes() []string {
	return workspaceurls.ResourceTypes(bundleconfig.WorkspaceURLPatterns())
}

func TestWorkspaceOpenResourceTypes(t *testing.T) {
	types := workspaceOpenResourceTypes()

	// Every bundle resource type with a single-ID page is listed, except
	// sql_warehouses, which is accepted as an alias of warehouses.
	for key := range bundleconfig.WorkspaceURLPatterns() {
		if key == "sql_warehouses" {
			assert.NotContains(t, types, key)
			continue
		}
		assert.Contains(t, types, key)
	}
	for _, key := range []string{"notebooks", "queries", "warehouses"} {
		assert.Contains(t, types, key)
	}

	// The alias builds the same URL as the type it stands for.
	viaAlias, err := workspaceurls.BuildResourceURL(bundleconfig.WorkspaceURLPatterns(), "https://myworkspace.databricks.com", "sql_warehouses", "wh-1", "")
	require.NoError(t, err)
	assert.Equal(t, "https://myworkspace.databricks.com/sql/warehouses/wh-1", viaAlias)

	// The alias resolves to the "warehouses" entry of workspaceurls, while bundle
	// summaries use the pattern that the sql_warehouses type declares. The two
	// must build the same URL.
	base, err := url.Parse("https://myworkspace.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, bundleconfig.WorkspaceURLPatterns()["sql_warehouses"].URL(*base, "wh-1"), viaAlias)
}

func TestWorkspaceOpenCommandCompletion(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Equal(t, workspaceOpenResourceTypes(), completions)
}

func TestWorkspaceOpenCommandCompletionSecondArg(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{"jobs"}, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Nil(t, completions)
}

func TestWorkspaceOpenCommandHelpText(t *testing.T) {
	cmd := newWorkspaceOpenCommand()

	assert.Contains(t, cmd.Long, "Supported resource types: "+strings.Join(workspaceOpenResourceTypes(), ", ")+".")
	assert.Contains(t, cmd.Long, "databricks experimental open jobs 123456789")
	assert.Contains(t, cmd.Long, "databricks experimental open notebooks /Users/user@example.com/my-notebook")
	assert.Contains(t, cmd.Long, "databricks experimental open registered_models catalog.schema.my_model")
	assert.Contains(t, cmd.Long, "databricks experimental open jobs 123456789 --url")
	assert.Contains(t, cmd.Long, "dot-separated name")

	flag := cmd.Flags().Lookup("url")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestWorkspaceOpenCommandOpensBrowserByDefault(t *testing.T) {
	originalResolveWorkspaceID := resolveWorkspaceID
	originalOpenWorkspaceURL := openWorkspaceURL
	t.Cleanup(func() {
		resolveWorkspaceID = originalResolveWorkspaceID
		openWorkspaceURL = originalOpenWorkspaceURL
	})

	resolveWorkspaceID = func(context.Context) (string, error) {
		return "", nil
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
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "Opening jobs 123 in the browser...")
}

func TestWorkspaceOpenCommandURLFlag(t *testing.T) {
	originalResolveWorkspaceID := resolveWorkspaceID
	originalOpenWorkspaceURL := openWorkspaceURL
	t.Cleanup(func() {
		resolveWorkspaceID = originalResolveWorkspaceID
		openWorkspaceURL = originalOpenWorkspaceURL
	})

	resolveWorkspaceID = func(context.Context) (string, error) {
		return "789", nil
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
	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123?w=789\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestWorkspaceOpenCommandWarnsWhenWorkspaceIDLookupFails(t *testing.T) {
	originalResolveWorkspaceID := resolveWorkspaceID
	originalOpenWorkspaceURL := openWorkspaceURL
	t.Cleanup(func() {
		resolveWorkspaceID = originalResolveWorkspaceID
		openWorkspaceURL = originalOpenWorkspaceURL
	})

	resolveWorkspaceID = func(context.Context) (string, error) {
		return "", errors.New("lookup failed")
	}

	openWorkspaceURL = func(ctx context.Context, targetURL string) error {
		return nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = log.NewContext(ctx, slog.New(handler.NewFriendlyHandler(stderr, &handler.Options{
		Level: log.LevelWarn,
	})))
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

	assert.Equal(t, "https://myworkspace.databricks.com/jobs/123\n", stdout.String())
	assert.Contains(t, stderr.String(), "Could not determine workspace ID: lookup failed")
}
