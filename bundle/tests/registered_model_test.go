package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func assertExpectedModel(t *testing.T, p *resources.RegisteredModel) {
	assert.Equal(t, "main", p.CatalogName)
	assert.Equal(t, "default", p.SchemaName)
	assert.Equal(t, "comment", p.Comment)
	assert.Equal(t, "account users", p.Grants[0].Principal)
	assert.Equal(t, "EXECUTE", p.Grants[0].Privileges[0])
}

func TestRegisteredModelDevelopment(t *testing.T) {
	b := loadTarget(t, "./registered_model", "development")
	assert.Len(t, b.Config.Resources.RegisteredModels, 1)
	assert.Equal(t, config.Development, b.Config.Bundle.Mode)

	p := b.Config.Resources.RegisteredModels["my_registered_model"]
	assert.Equal(t, "my-dev-model", p.Name)
	assertExpectedModel(t, p)
}

func TestRegisteredModelStaging(t *testing.T) {
	b := loadTarget(t, "./registered_model", "staging")
	assert.Len(t, b.Config.Resources.RegisteredModels, 1)

	p := b.Config.Resources.RegisteredModels["my_registered_model"]
	assert.Equal(t, "my-staging-model", p.Name)
	assertExpectedModel(t, p)
}

func TestRegisteredModelProduction(t *testing.T) {
	b := loadTarget(t, "./registered_model", "production")
	assert.Len(t, b.Config.Resources.RegisteredModels, 1)

	p := b.Config.Resources.RegisteredModels["my_registered_model"]
	assert.Equal(t, "my-prod-model", p.Name)
	assertExpectedModel(t, p)
}
