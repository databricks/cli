package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestValidateDirectOnlyResourcesDirectEngineReturnsNil(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateDirectOnlyResources(engine.EngineDirect))
	assert.Empty(t, diags)
}

func TestValidateDirectOnlyResourcesTerraformEngineNoDirectOnlyReturnsNil(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateDirectOnlyResources(engine.EngineTerraform))
	assert.Empty(t, diags)
}

func TestValidateDirectOnlyResourcesTerraformEngineDirectOnlyEmitsError(t *testing.T) {
	cases := []struct {
		name            string
		bundle          *bundle.Bundle
		expectedSummary string
		expectedDetail  string
	}{
		{
			name: "catalogs",
			bundle: &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Catalogs: map[string]*resources.Catalog{
							"my_catalog": {},
						},
					},
				},
			},
			expectedSummary: "Catalog resources are only supported with direct deployment mode",
			expectedDetail: "Catalog resources require direct deployment mode. " +
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use catalog resources.\n" +
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
		},
		{
			name: "external_locations",
			bundle: &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						ExternalLocations: map[string]*resources.ExternalLocation{
							"my_location": {},
						},
					},
				},
			},
			expectedSummary: "External Location resources are only supported with direct deployment mode",
			expectedDetail: "External Location resources require direct deployment mode. " +
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use external_location resources.\n" +
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
		},
		{
			name: "vector_search_endpoints",
			bundle: &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						VectorSearchEndpoints: map[string]*resources.VectorSearchEndpoint{
							"my_endpoint": {},
						},
					},
				},
			},
			expectedSummary: "Vector Search Endpoint resources are only supported with direct deployment mode",
			expectedDetail: "Vector Search Endpoint resources require direct deployment mode. " +
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use vector_search_endpoint resources.\n" +
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := bundle.Apply(t.Context(), tc.bundle, mutator.ValidateDirectOnlyResources(engine.EngineTerraform))
			if assert.Len(t, diags, 1) {
				assert.Equal(t, tc.expectedSummary, diags[0].Summary)
				assert.Equal(t, tc.expectedDetail, diags[0].Detail)
			}
		})
	}
}
