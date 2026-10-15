package config_tests

import (
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func assertExpected(t *testing.T, p *resources.ModelServingEndpoint) {
	// After ModelServingEndpointFixups mutator, served_models is converted to served_entities
	assert.Nil(t, p.Config.ServedModels, "served_models should be nil after conversion")
	assert.Len(t, p.Config.ServedEntities, 1, "served_entities should have one entity")
	assert.Equal(t, "model-name", utils.Deref(p.Config.ServedEntities[0].EntityName))
	assert.Equal(t, "1", utils.Deref(p.Config.ServedEntities[0].EntityVersion))

	assert.Equal(t, "model-name-1", utils.Deref(p.Config.TrafficConfig.Routes[0].ServedModelName))
	assert.Equal(t, 100, utils.Deref(p.Config.TrafficConfig.Routes[0].TrafficPercentage))
	assert.Equal(t, resources.ModelServingEndpointPermission{
		GroupName: "users",
		Level:     "CAN_QUERY",
	}, p.Permissions[0])
}

func TestModelServingEndpointDevelopment(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "development")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)
	assert.Equal(t, config.Development, b.Config.Bundle.Mode)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-dev-endpoint", utils.Deref(p.Name))
	assertExpected(t, p)
}

func TestModelServingEndpointStaging(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "staging")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-staging-endpoint", utils.Deref(p.Name))
	assertExpected(t, p)
}

func TestModelServingEndpointProduction(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "production")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-prod-endpoint", utils.Deref(p.Name))
	assertExpected(t, p)
}
