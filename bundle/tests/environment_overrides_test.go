package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentOverridesWorkspaceDev(t *testing.T) {
	b := loadTarget(t, "./environment_overrides/workspace", "development")
	assert.Equal(t, "https://development.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestEnvironmentOverridesWorkspaceStaging(t *testing.T) {
	b := loadTarget(t, "./environment_overrides/workspace", "staging")
	assert.Equal(t, "https://staging.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestEnvironmentOverridesResourcesDev(t *testing.T) {
	b := loadTarget(t, "./environment_overrides/resources", "development")
	assert.Equal(t, "base job", b.Config.Resources.Jobs["job1"].Name)

	// Base values are preserved in the development environment.
	assert.True(t, b.Config.Resources.Pipelines["boolean1"].Photon)
	assert.False(t, b.Config.Resources.Pipelines["boolean2"].Photon)
}

func TestEnvironmentOverridesResourcesStaging(t *testing.T) {
	b := loadTarget(t, "./environment_overrides/resources", "staging")
	assert.Equal(t, "staging job", b.Config.Resources.Jobs["job1"].Name)

	// Override values are applied in the staging environment.
	assert.False(t, b.Config.Resources.Pipelines["boolean1"].Photon)
	assert.True(t, b.Config.Resources.Pipelines["boolean2"].Photon)
}
