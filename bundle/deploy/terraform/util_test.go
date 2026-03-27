package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResourcesStateWithNoFile(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "whatever",
				Terraform: &config.Terraform{
					ExecPath: "terraform",
				},
			},
		},
	}
	state, err := ParseResourcesState(t.Context(), b)
	assert.NoError(t, err)
	assert.Equal(t, ExportedResourcesMap(nil), state)
}

func TestParseResourcesStateWithExistingStateFile(t *testing.T) {
	ctx := t.Context()
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "whatever",
				Terraform: &config.Terraform{
					ExecPath: "terraform",
				},
			},
		},
	}
	data := []byte(`{
		"version": 4,
		"unknown_field": "hello",
		"resources": [
		{
			"mode": "managed",
			"type": "databricks_pipeline",
			"name": "test_pipeline",
			"provider": "provider[\"registry.terraform.io/databricks/databricks\"]",
			"instances": [
			  {
				"schema_version": 0,
				"attributes": {
				  "allow_duplicate_names": false,
				  "catalog": null,
				  "channel": "CURRENT",
				  "cluster": [],
				  "random_field": "random_value",
				  "configuration": {
					"bundle.sourcePath": "/Workspace//Users/user/.bundle/test/dev/files/src"
				  },
				  "continuous": false,
				  "development": true,
				  "edition": "ADVANCED",
				  "filters": [],
				  "id": "123",
				  "library": [],
				  "name": "test_pipeline",
				  "notification": [],
				  "photon": false,
				  "serverless": false,
				  "storage": "dbfs:/123456",
				  "target": "test_dev",
				  "timeouts": null,
				  "url": "https://test.com"
				},
				"sensitive_attributes": []
			  }
			]
		  }
		]
	}`)
	_, localPath := b.StateFilenameTerraform(ctx)
	err := os.MkdirAll(filepath.Dir(localPath), 0o700)
	assert.NoError(t, err)
	err = os.WriteFile(localPath, data, 0o600)
	assert.NoError(t, err)
	state, err := parseResourcesState(ctx, localPath)
	assert.NoError(t, err)
	expected := ExportedResourcesMap{
		"resources.pipelines.test_pipeline": {ID: "123"},
	}
	assert.Equal(t, expected, state)
}

func TestParseResourcesStateSecretScopeWithAcls(t *testing.T) {
	ctx := t.Context()
	data := []byte(`{
		"version": 4,
		"resources": [
		{
			"mode": "managed",
			"type": "databricks_secret_scope",
			"name": "my_scope",
			"instances": [{"attributes": {"id": "123", "name": "actual-scope-name"}}]
		},
		{
			"mode": "managed",
			"type": "databricks_secret_acl",
			"name": "secret_acl_my_scope_0",
			"instances": [{"attributes": {"id": "actual-scope-name|||user@example.com"}}]
		},
		{
			"mode": "managed",
			"type": "databricks_secret_acl",
			"name": "secret_acl_my_scope_1",
			"instances": [{"attributes": {"id": "actual-scope-name|||data-team"}}]
		}
		]
	}`)
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	state, err := parseResourcesState(ctx, path)
	require.NoError(t, err)

	assert.Equal(t, ExportedResourcesMap{
		"resources.secret_scopes.my_scope":             {ID: "actual-scope-name"},
		"resources.secret_scopes.my_scope.permissions": {ID: "actual-scope-name"},
	}, state)
}

func TestParseResourcesStateSecretScopeWithoutAcls(t *testing.T) {
	ctx := t.Context()
	data := []byte(`{
		"version": 4,
		"resources": [
		{
			"mode": "managed",
			"type": "databricks_secret_scope",
			"name": "my_scope",
			"instances": [{"attributes": {"id": "123", "name": "my-scope-name"}}]
		}
		]
	}`)
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	state, err := parseResourcesState(ctx, path)
	require.NoError(t, err)

	// No ACLs → no permissions entry; migrate.go fixup handles this case.
	assert.Equal(t, ExportedResourcesMap{
		"resources.secret_scopes.my_scope": {ID: "my-scope-name"},
	}, state)
}
