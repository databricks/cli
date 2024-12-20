package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, &resourcesState{Version: SupportedStateVersion}, state)
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
	cacheDir, err := Dir(ctx, b)
	assert.NoError(t, err)
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
	err = os.WriteFile(filepath.Join(cacheDir, TerraformStateFileName), data, os.ModePerm)
	assert.NoError(t, err)
	state, err := ParseResourcesState(ctx, b)
	assert.NoError(t, err)
	expected := &resourcesState{
		Version: 4,
		Resources: []stateResource{
			{
				Mode: "managed",
				Type: "databricks_pipeline",
				Name: "test_pipeline",
				Instances: []stateResourceInstance{
					{Attributes: stateInstanceAttributes{ID: "123", Name: "test_pipeline"}},
				},
			},
		},
	}
	assert.Equal(t, expected, state)
}
