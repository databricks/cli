package resourcemutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelServingEndpointFixups(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "test_endpoint",
							Config: &serving.EndpointCoreConfigInput{
								ServedModels: []serving.ServedModelInput{
									{
										ModelName:          "model1",
										ModelVersion:       "1",
										ScaleToZeroEnabled: true,
										WorkloadSize:       "Medium",
										EnvironmentVars: map[string]string{
											"KEY": "value",
										},
									},
									{
										ModelName:    "model2",
										ModelVersion: "2",
										// WorkloadSize not specified - should get default "Small"
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(ctx, b, resourcemutator.ModelServingEndpointFixups())
	require.NoError(t, diags.Error())

	// Verify warning is emitted
	require.Len(t, diags, 1)
	assert.Equal(t, "Using served_models is deprecated", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, "Please use served_entities instead")

	endpoint := b.Config.Resources.ModelServingEndpoints["endpoint1"]

	// Verify ServedModels is cleared
	assert.Nil(t, endpoint.Config.ServedModels)

	// Verify ServedEntities is populated
	require.Len(t, endpoint.Config.ServedEntities, 2)

	// Check first entity
	assert.Equal(t, "model1", endpoint.Config.ServedEntities[0].EntityName)
	assert.Equal(t, "1", endpoint.Config.ServedEntities[0].EntityVersion)
	assert.True(t, endpoint.Config.ServedEntities[0].ScaleToZeroEnabled)
	assert.Equal(t, "Medium", endpoint.Config.ServedEntities[0].WorkloadSize)
	assert.Equal(t, map[string]string{"KEY": "value"}, endpoint.Config.ServedEntities[0].EnvironmentVars)

	// Check second entity - should have default "Small" workload_size
	assert.Equal(t, "model2", endpoint.Config.ServedEntities[1].EntityName)
	assert.Equal(t, "2", endpoint.Config.ServedEntities[1].EntityVersion)
	assert.Equal(t, "Small", endpoint.Config.ServedEntities[1].WorkloadSize)
}

func TestModelServingEndpointFixups_ErrorOnBothServedModelsAndServedEntities(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "test_endpoint",
							Config: &serving.EndpointCoreConfigInput{
								ServedModels: []serving.ServedModelInput{
									{
										ModelName:    "model1",
										ModelVersion: "1",
									},
								},
								ServedEntities: []serving.ServedEntityInput{
									{
										EntityName:    "entity1",
										EntityVersion: "1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(ctx, b, resourcemutator.ModelServingEndpointFixups())
	require.Error(t, diags.Error())

	assert.Len(t, diags, 1)
	assert.Equal(t, "Cannot use both served_models and served_entities", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, "cannot specify both served_models and served_entities")
}

func TestModelServingEndpointFixups_NoConfig(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "test_endpoint",
							// No Config specified
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(ctx, b, resourcemutator.ModelServingEndpointFixups())
	require.NoError(t, diags.Error())
}
