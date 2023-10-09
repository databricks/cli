package config_tests

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func assertExpected(t *testing.T, p *resources.ModelServingEndpoint) {
	assert.Equal(t, "model_serving_endpoint/databricks.yml", filepath.ToSlash(p.LocalConfigFilePath))
	assert.Equal(t, "model-name", p.Config.ServedModels[0].ModelName)
	assert.Equal(t, "1", p.Config.ServedModels[0].ModelVersion)
	assert.Equal(t, "model-name-1", p.Config.TrafficConfig.Routes[0].ServedModelName)
	assert.Equal(t, 100, p.Config.TrafficConfig.Routes[0].TrafficPercentage)
	assert.Equal(t, "users", p.Permissions[0].GroupName)
	assert.Equal(t, "CAN_QUERY", p.Permissions[0].Level)
}

func TestModelServingEndpointDevelopment(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "development")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)
	assert.Equal(t, b.Config.Bundle.Mode, config.Development)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-dev-endpoint", p.Name)
	assertExpected(t, p)
}

func TestModelServingEndpointStaging(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "staging")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-staging-endpoint", p.Name)
	assertExpected(t, p)
}

func TestModelServingEndpointProduction(t *testing.T) {
	b := loadTarget(t, "./model_serving_endpoint", "production")
	assert.Len(t, b.Config.Resources.ModelServingEndpoints, 1)

	p := b.Config.Resources.ModelServingEndpoints["my_model_serving_endpoint"]
	assert.Equal(t, "my-prod-endpoint", p.Name)
	assertExpected(t, p)
}
