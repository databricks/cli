package terraform

import (
	"context"
	"os"
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
	state, err := ParseResourcesState(context.Background(), b)
	assert.NoError(t, err)
	assert.Equal(t, ExportedResourcesMap(nil), state)
}

func TestParseResourcesStateWithExistingStateFile(t *testing.T) {
	ctx := context.Background()
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
	path, err := b.StateLocalPath(ctx)
	require.NoError(t, err)
	err = os.WriteFile(path, data, os.ModePerm)
	assert.NoError(t, err)
	state, err := ParseResourcesState(ctx, b)
	assert.NoError(t, err)
	expected := ExportedResourcesMap{
		"pipelines": map[string]ResourceState{
			"test_pipeline": {ID: "123"},
		},
	}
	assert.Equal(t, expected, state)
}
