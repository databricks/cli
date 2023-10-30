package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestBundlePermissions(t *testing.T) {
	b := load(t, "./bundle_permissions")
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "bot@company.com"})
}

func TestBundlePermissionsDevTarget(t *testing.T) {
	b := loadTarget(t, "./bundle_permissions", "development")
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "bot@company.com"})
}
