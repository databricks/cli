package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newParseUcm builds a minimal *ucm.Ucm whose WorkingDir resolves under a
// temp dir, mirroring how the migrate verb will see a fresh project.
func newParseUcm(t *testing.T) *ucm.Ucm {
	t.Helper()
	root := t.TempDir()
	cfg := config.Root{}
	cfg.Ucm.Name = "parse-test"
	cfg.Ucm.Target = "dev"
	return &ucm.Ucm{RootPath: root, Config: cfg}
}

func TestParseResourcesStateWithNoFile(t *testing.T) {
	u := newParseUcm(t)
	state, err := ParseResourcesState(t.Context(), u)
	assert.NoError(t, err)
	assert.Equal(t, ExportedResourcesMap(nil), state)
}

func TestParseResourcesStateWithExistingStateFile(t *testing.T) {
	ctx := t.Context()
	u := newParseUcm(t)
	workingDir, err := WorkingDir(u)
	require.NoError(t, err)

	data := []byte(`{
		"version": 4,
		"unknown_field": "hello",
		"resources": [
		{
			"mode": "managed",
			"type": "databricks_catalog",
			"name": "sales",
			"provider": "provider[\"registry.terraform.io/databricks/databricks\"]",
			"instances": [
			  {
				"schema_version": 0,
				"attributes": {
				  "id": "sales_prod",
				  "name": "sales_prod",
				  "comment": "sales data"
				},
				"sensitive_attributes": []
			  }
			]
		  },
		  {
			"mode": "managed",
			"type": "databricks_schema",
			"name": "raw",
			"instances": [{"attributes": {"id": "sales_prod.raw", "name": "raw"}}]
		  }
		]
	}`)
	localPath := filepath.Join(workingDir, "terraform.tfstate")
	require.NoError(t, os.MkdirAll(filepath.Dir(localPath), 0o700))
	require.NoError(t, os.WriteFile(localPath, data, 0o600))

	state, err := parseResourcesState(ctx, localPath)
	require.NoError(t, err)
	expected := ExportedResourcesMap{
		"resources.catalogs.sales": {ID: "sales_prod"},
		"resources.schemas.raw":    {ID: "sales_prod.raw"},
	}
	assert.Equal(t, expected, state)
}

func TestParseResourcesStateSkipsDataSources(t *testing.T) {
	ctx := t.Context()
	data := []byte(`{
		"version": 4,
		"resources": [
		{
			"mode": "data",
			"type": "databricks_catalog",
			"name": "lookup",
			"instances": [{"attributes": {"id": "ignored"}}]
		},
		{
			"mode": "managed",
			"type": "databricks_catalog",
			"name": "sales",
			"instances": [{"attributes": {"id": "sales_prod"}}]
		}
		]
	}`)
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	state, err := parseResourcesState(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, ExportedResourcesMap{
		"resources.catalogs.sales": {ID: "sales_prod"},
	}, state)
}

func TestParseResourcesStateSkipsUnknownTypes(t *testing.T) {
	// Defensive: a tfstate carrying a type ucm doesn't model (e.g. an
	// upstream-only databricks_job) is logged-and-skipped rather than
	// hard-failing the migrate verb.
	ctx := t.Context()
	data := []byte(`{
		"version": 4,
		"resources": [
		{
			"mode": "managed",
			"type": "databricks_job",
			"name": "etl",
			"instances": [{"attributes": {"id": "999"}}]
		},
		{
			"mode": "managed",
			"type": "databricks_catalog",
			"name": "sales",
			"instances": [{"attributes": {"id": "sales_prod"}}]
		}
		]
	}`)
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	state, err := parseResourcesState(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, ExportedResourcesMap{
		"resources.catalogs.sales": {ID: "sales_prod"},
	}, state)
}

func TestParseResourcesStateRejectsUnsupportedVersion(t *testing.T) {
	ctx := t.Context()
	data := []byte(`{
		"version": 3,
		"resources": []
	}`)
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	state, err := parseResourcesState(ctx, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported deployment state version: 3")
	assert.Nil(t, state)
}

func TestParseResourcesStateRejectsMalformedJSON(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := parseResourcesState(ctx, path)
	require.Error(t, err)
}
