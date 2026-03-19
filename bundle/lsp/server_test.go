package lsp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/lsp"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexResources(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_etl_job:
      name: "ETL Job"
  pipelines:
    data_pipeline:
      name: "Pipeline"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	doc := &lsp.Document{
		URI:     "file:///test.yml",
		Content: yaml,
		Lines:   strings.Split(yaml, "\n"),
		Value:   v,
	}

	entries := lsp.IndexResources(doc)
	require.Len(t, entries, 2)

	// Verify first entry (jobs.my_etl_job).
	assert.Equal(t, "jobs", entries[0].Type)
	assert.Equal(t, "my_etl_job", entries[0].Key)
	assert.Equal(t, "resources.jobs.my_etl_job", entries[0].Path)
	// Key should span the length of "my_etl_job".
	assert.Equal(t, entries[0].KeyRange.Start.Character+len("my_etl_job"), entries[0].KeyRange.End.Character)

	// Verify second entry (pipelines.data_pipeline).
	assert.Equal(t, "pipelines", entries[1].Type)
	assert.Equal(t, "data_pipeline", entries[1].Key)
	assert.Equal(t, "resources.pipelines.data_pipeline", entries[1].Path)
}

func TestIndexResourcesInvalidYAML(t *testing.T) {
	doc := &lsp.Document{
		URI:   "file:///bad.yml",
		Value: dyn.InvalidValue,
	}

	entries := lsp.IndexResources(doc)
	assert.Nil(t, entries)
}

func TestIndexResourcesNoResources(t *testing.T) {
	yaml := `
bundle:
  name: "test"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	doc := &lsp.Document{
		URI:   "file:///test.yml",
		Value: v,
	}

	entries := lsp.IndexResources(doc)
	assert.Nil(t, entries)
}

func TestPositionInRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      lsp.Position
		r        lsp.Range
		expected bool
	}{
		{
			name:     "inside range",
			pos:      lsp.Position{Line: 3, Character: 5},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: true,
		},
		{
			name:     "at start of range",
			pos:      lsp.Position{Line: 3, Character: 4},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: true,
		},
		{
			name:     "at end of range (exclusive)",
			pos:      lsp.Position{Line: 3, Character: 14},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: false,
		},
		{
			name:     "before range",
			pos:      lsp.Position{Line: 3, Character: 2},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: false,
		},
		{
			name:     "after range",
			pos:      lsp.Position{Line: 3, Character: 20},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: false,
		},
		{
			name:     "wrong line above",
			pos:      lsp.Position{Line: 2, Character: 5},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: false,
		},
		{
			name:     "wrong line below",
			pos:      lsp.Position{Line: 4, Character: 5},
			r:        lsp.Range{Start: lsp.Position{Line: 3, Character: 4}, End: lsp.Position{Line: 3, Character: 14}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, lsp.PositionInRange(tt.pos, tt.r))
		})
	}
}

func TestBuildResourceURL(t *testing.T) {
	host := "https://my-workspace.databricks.com"

	tests := []struct {
		resourceType string
		id           string
		expected     string
	}{
		{"jobs", "123", host + "/jobs/123"},
		{"pipelines", "abc-def", host + "/pipelines/abc-def"},
		{"dashboards", "d1", host + "/dashboardsv3/d1/published"},
		{"model_serving_endpoints", "ep1", host + "/ml/endpoints/ep1"},
		{"experiments", "exp1", host + "/ml/experiments/exp1"},
		{"models", "m1", host + "/ml/models/m1"},
		{"clusters", "c1", host + "/compute/clusters/c1"},
		{"apps", "a1", host + "/apps/a1"},
		{"alerts", "al1", host + "/sql/alerts-v2/al1"},
		{"sql_warehouses", "sw1", host + "/sql/warehouses/sw1"},
		{"quality_monitors", "qm1", host + "/explore/data/qm1"},
		{"secret_scopes", "ss1", host + "/secrets/scopes/ss1"},
		{"unknown_type", "x1", host},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			assert.Equal(t, tt.expected, lsp.BuildResourceURL(host, tt.resourceType, tt.id))
		})
	}
}

func TestBuildResourceURLEmptyInputs(t *testing.T) {
	assert.Equal(t, "", lsp.BuildResourceURL("", "jobs", "123"))
	assert.Equal(t, "", lsp.BuildResourceURL("https://host.com", "jobs", ""))
}

func TestBuildResourceURLTrailingSlash(t *testing.T) {
	assert.Equal(t, "https://host.com/jobs/123", lsp.BuildResourceURL("https://host.com/", "jobs", "123"))
}

func TestLoadWorkspaceHost(t *testing.T) {
	yaml := `
workspace:
  host: "https://my-workspace.databricks.com"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	assert.Equal(t, "https://my-workspace.databricks.com", lsp.LoadWorkspaceHost(v))
}

func TestLoadWorkspaceHostWithInterpolation(t *testing.T) {
	yaml := `
workspace:
  host: "${var.host}"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	assert.Equal(t, "", lsp.LoadWorkspaceHost(v))
}

func TestLoadWorkspaceHostMissing(t *testing.T) {
	yaml := `
bundle:
  name: "test"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	assert.Equal(t, "", lsp.LoadWorkspaceHost(v))
}

func TestLoadTarget(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			name: "default target marked",
			yaml: `
targets:
  dev:
    default: true
  prod:
    workspace:
      host: "https://prod.databricks.com"
`,
			expected: "dev",
		},
		{
			name: "no default returns first",
			yaml: `
targets:
  staging:
    workspace:
      host: "https://staging.databricks.com"
  prod:
    workspace:
      host: "https://prod.databricks.com"
`,
			expected: "staging",
		},
		{
			name: "no targets section",
			yaml: `
bundle:
  name: "test"
`,
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, lsp.LoadTarget(v))
		})
	}
}

func TestURIToPathRoundTrip(t *testing.T) {
	// Test that PathToURI and URIToPath are inverses of each other.
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "databricks.yml")
	uri := lsp.PathToURI(path)
	assert.Equal(t, path, lsp.URIToPath(uri))
}

func TestDocumentStoreOpenGetClose(t *testing.T) {
	store := lsp.NewDocumentStore()

	assert.Nil(t, store.Get("file:///test.yml"))

	store.Open("file:///test.yml", 1, "key: value")
	doc := store.Get("file:///test.yml")
	require.NotNil(t, doc)
	assert.Equal(t, 1, doc.Version)
	assert.Equal(t, "key: value", doc.Content)
	assert.True(t, doc.Value.IsValid())

	store.Close("file:///test.yml")
	assert.Nil(t, store.Get("file:///test.yml"))
}

func TestDocumentStoreChange(t *testing.T) {
	store := lsp.NewDocumentStore()

	store.Open("file:///test.yml", 1, "key: value")
	store.Change("file:///test.yml", 2, "key: updated")

	doc := store.Get("file:///test.yml")
	require.NotNil(t, doc)
	assert.Equal(t, 2, doc.Version)
	assert.Equal(t, "key: updated", doc.Content)
	assert.True(t, doc.Value.IsValid())
}

func TestDocumentStoreParseInvalidYAML(t *testing.T) {
	store := lsp.NewDocumentStore()
	store.Open("file:///bad.yml", 1, "{{{{invalid yaml")
	doc := store.Get("file:///bad.yml")
	require.NotNil(t, doc)
	assert.False(t, doc.Value.IsValid())
}

func TestLoadResourceStateDirectEngine(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".databricks", "bundle", "dev")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	stateJSON := `{
		"state_version": 1,
		"state": {
			"resources.jobs.etl": {"__id__": "111"},
			"resources.pipelines.dlt": {"__id__": "222"}
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "resources.json"), []byte(stateJSON), 0o644))

	result := lsp.LoadResourceState(tmpDir, "dev")
	assert.Equal(t, "111", result["resources.jobs.etl"].ID)
	assert.Equal(t, "222", result["resources.pipelines.dlt"].ID)
}

func TestLoadResourceStateNoState(t *testing.T) {
	result := lsp.LoadResourceState("/nonexistent", "dev")
	assert.Empty(t, result)
}

func TestLoadAllTargets(t *testing.T) {
	yaml := `
targets:
  dev:
    default: true
  staging:
    workspace:
      host: "https://staging.databricks.com"
  prod:
    workspace:
      host: "https://prod.databricks.com"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	targets := lsp.LoadAllTargets(v)
	require.Len(t, targets, 3)
	assert.Equal(t, "dev", targets[0])
	assert.Equal(t, "staging", targets[1])
	assert.Equal(t, "prod", targets[2])
}

func TestLoadAllTargetsNoTargets(t *testing.T) {
	yaml := `
bundle:
  name: "test"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	targets := lsp.LoadAllTargets(v)
	assert.Nil(t, targets)
}

func TestLoadTargetWorkspaceHost(t *testing.T) {
	yaml := `
workspace:
  host: "https://default.databricks.com"
targets:
  dev:
    workspace:
      host: "https://dev.databricks.com"
  prod:
    workspace:
      host: "https://prod.databricks.com"
  staging: {}
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	assert.Equal(t, "https://dev.databricks.com", lsp.LoadTargetWorkspaceHost(v, "dev"))
	assert.Equal(t, "https://prod.databricks.com", lsp.LoadTargetWorkspaceHost(v, "prod"))
	// staging has no override, falls back to root.
	assert.Equal(t, "https://default.databricks.com", lsp.LoadTargetWorkspaceHost(v, "staging"))
}

func TestPathToURI(t *testing.T) {
	assert.Equal(t, "file:///home/user/file.yml", lsp.PathToURI("/home/user/file.yml"))
}
