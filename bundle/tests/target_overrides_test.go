package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetOverridesWorkspaceDev(t *testing.T) {
	b := loadTarget(t, "./target_overrides/workspace", "development")
	assert.Equal(t, "https://development.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestTargetOverridesWorkspaceStaging(t *testing.T) {
	b := loadTarget(t, "./target_overrides/workspace", "staging")
	assert.Equal(t, "https://staging.acme.cloud.databricks.com/", b.Config.Workspace.Host)
}

func TestTargetOverridesResourcesDev(t *testing.T) {
	b := loadTarget(t, "./target_overrides/resources", "development")
	assert.Equal(t, "base job", b.Config.Resources.Jobs["job1"].Name)
}

func TestTargetOverridesResourcesStaging(t *testing.T) {
	b := loadTarget(t, "./target_overrides/resources", "staging")
	assert.Equal(t, "staging job", b.Config.Resources.Jobs["job1"].Name)
}
