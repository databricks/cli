package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentOverridesWorkspaceDev(t *testing.T) {
	b := loadEnvironment(t, "./environment_overrides/workspace", "development")
	assert.Equal(t, "https://development.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestEnvironmentOverridesWorkspaceStaging(t *testing.T) {
	b := loadEnvironment(t, "./environment_overrides/workspace", "staging")
	assert.Equal(t, "https://staging.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestEnvironmentOverridesResourcesDev(t *testing.T) {
	b := loadEnvironment(t, "./environment_overrides/resources", "development")
	assert.Equal(t, "base job", b.Config.Resources.Jobs["job1"].Name)

	// Base values are preserved in the development environment.
	assert.Equal(t, true, b.Config.Resources.Pipelines["boolean1"].Photon)
	assert.Equal(t, false, b.Config.Resources.Pipelines["boolean2"].Photon)
}

func TestEnvironmentOverridesResourcesStaging(t *testing.T) {
	b := loadEnvironment(t, "./environment_overrides/resources", "staging")
	assert.Equal(t, "staging job", b.Config.Resources.Jobs["job1"].Name)

	// Overrides are only applied if they are not zero-valued.
	// This means that in its current form, we cannot override a true value with a false value.
	// Note: this is not desirable and will be addressed by representing our configuration
	// in a different structure (e.g. with cty), instead of Go structs.
	assert.Equal(t, true, b.Config.Resources.Pipelines["boolean1"].Photon)
	assert.Equal(t, true, b.Config.Resources.Pipelines["boolean2"].Photon)
}
