package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestApps(t *testing.T) {
	b := load(t, "./apps")
	assert.Equal(t, "apps", b.Config.Bundle.Name)

	diags := bundle.Apply(context.Background(), b,
		bundle.Seq(
			mutator.SetVariables(),
			mutator.ResolveVariableReferences("variables"),
		))
	assert.Empty(t, diags)

	app := b.Config.Resources.Apps["my_app"]
	assert.Equal(t, "my-app", app.Name)
	assert.Equal(t, "My App", app.Description)
	assert.Equal(t, []any{"python", "app.py"}, app.Config["command"])
	assert.Equal(t, []any{map[string]any{"name": "SOME_ENV_VARIABLE", "value": "Some value"}}, app.Config["env"])

	assert.Len(t, app.Resources, 2)
	assert.Equal(t, "1234", app.Resources[0].SqlWarehouse.Id)
	assert.Equal(t, "CAN_USE", string(app.Resources[0].SqlWarehouse.Permission))
	assert.Equal(t, "5678", app.Resources[1].Job.Id)
	assert.Equal(t, "CAN_MANAGE_RUN", string(app.Resources[1].Job.Permission))
}

func TestAppsOverride(t *testing.T) {
	b := loadTarget(t, "./apps", "development")
	assert.Equal(t, "apps", b.Config.Bundle.Name)

	diags := bundle.Apply(context.Background(), b,
		bundle.Seq(
			mutator.SetVariables(),
			mutator.ResolveVariableReferences("variables"),
		))
	assert.Empty(t, diags)
	app := b.Config.Resources.Apps["my_app"]
	assert.Equal(t, "my-app", app.Name)
	assert.Equal(t, "My App", app.Description)
	assert.Equal(t, []any{"python", "dev.py"}, app.Config["command"])
	assert.Equal(t, []any{map[string]any{"name": "SOME_ENV_VARIABLE_2", "value": "Some value 2"}}, app.Config["env"])

	assert.Len(t, app.Resources, 3)
	assert.Equal(t, "1234", app.Resources[0].SqlWarehouse.Id)
	assert.Equal(t, "CAN_MANAGE", string(app.Resources[0].SqlWarehouse.Permission))
	assert.Equal(t, "5678", app.Resources[1].Job.Id)
	assert.Equal(t, "CAN_MANAGE", string(app.Resources[1].Job.Permission))
	assert.Equal(t, "key", app.Resources[2].Secret.Key)
	assert.Equal(t, "scope", app.Resources[2].Secret.Scope)
	assert.Equal(t, "CAN_USE", string(app.Resources[2].Secret.Permission))
}
