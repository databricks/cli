package snapshot_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/snapshot"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func bundleWithPermissions(currentUser string, perms []resources.Permission) *bundle.Bundle {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{},
			},
			Permissions: perms,
		},
	}
	b.Config.Workspace.CurrentUser.User = &iam.User{UserName: currentUser}
	return b
}

func TestBuildACLCurrentUserOnly(t *testing.T) {
	b := bundleWithPermissions("alice@example.com", nil)

	acl := snapshot.BuildACL(b)

	assert.Equal(t, []snapshot.ACLEntry{
		{UserName: "alice@example.com", PermissionLevel: "CAN_READ"},
	}, acl)
}

func TestBuildACLWithTopLevelPermissions(t *testing.T) {
	perms := []resources.Permission{
		{Level: "CAN_VIEW", UserName: "bob@example.com"},
		{Level: "CAN_MANAGE", GroupName: "devs"},
		{Level: "CAN_RUN", ServicePrincipalName: "sp-123"},
	}
	b := bundleWithPermissions("alice@example.com", perms)

	acl := snapshot.BuildACL(b)

	assert.Equal(t, []snapshot.ACLEntry{
		{UserName: "alice@example.com", PermissionLevel: "CAN_READ"},
		{UserName: "bob@example.com", PermissionLevel: "CAN_READ"},
		{GroupName: "devs", PermissionLevel: "CAN_READ"},
		{ServicePrincipalName: "sp-123", PermissionLevel: "CAN_READ"},
	}, acl)
}
