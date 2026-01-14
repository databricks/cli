package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSecretScopeWithPermissions(t *testing.T) {
	src := resources.SecretScope{
		Name: "my_scope",
		Permissions: []resources.SecretScopePermission{
			{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelWrite},
			{GroupName: "data-team", Level: resources.SecretScopePermissionLevelRead},
			{ServicePrincipalName: "sp-uuid", Level: resources.SecretScopePermissionLevelManage},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "my_scope", vin, out)
	require.NoError(t, err)

	// Verify ACL count
	assert.Len(t, out.SecretAcl, 3)

	// Verify first ACL depends only on scope
	acl0 := out.SecretAcl["secret_acl_my_scope_0"].(*resourceSecretAcl)
	assert.Equal(t, "user@example.com", acl0.Principal)
	assert.Equal(t, "WRITE", acl0.Permission)
	assert.Equal(t, []string{"databricks_secret_scope.my_scope"}, acl0.DependsOn)

	// Verify second ACL depends on scope + first ACL (sequential execution)
	acl1 := out.SecretAcl["secret_acl_my_scope_1"].(*resourceSecretAcl)
	assert.Equal(t, "data-team", acl1.Principal)
	assert.Equal(t, "READ", acl1.Permission)
	assert.Equal(t, []string{
		"databricks_secret_scope.my_scope",
		"databricks_secret_acl.secret_acl_my_scope_0",
	}, acl1.DependsOn)

	// Verify third ACL depends on scope + second ACL (sequential execution)
	acl2 := out.SecretAcl["secret_acl_my_scope_2"].(*resourceSecretAcl)
	assert.Equal(t, "sp-uuid", acl2.Principal)
	assert.Equal(t, "MANAGE", acl2.Permission)
	assert.Equal(t, []string{
		"databricks_secret_scope.my_scope",
		"databricks_secret_acl.secret_acl_my_scope_1",
	}, acl2.DependsOn)
}

func TestConvertSecretScopeSinglePermission(t *testing.T) {
	src := resources.SecretScope{
		Name: "single_scope",
		Permissions: []resources.SecretScopePermission{
			{UserName: "user@example.com", Level: resources.SecretScopePermissionLevelManage},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "single_scope", vin, out)
	require.NoError(t, err)

	// Single ACL should only depend on scope (no chaining needed)
	assert.Len(t, out.SecretAcl, 1)
	acl := out.SecretAcl["secret_acl_single_scope_0"].(*resourceSecretAcl)
	assert.Equal(t, "user@example.com", acl.Principal)
	assert.Equal(t, []string{"databricks_secret_scope.single_scope"}, acl.DependsOn)
}

func TestConvertSecretScopeNoPermissions(t *testing.T) {
	src := resources.SecretScope{
		Name: "no_permissions_scope",
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "no_permissions_scope", vin, out)
	require.NoError(t, err)

	// No ACLs should be created
	assert.Len(t, out.SecretAcl, 0)

	// But the scope should still be created
	assert.Contains(t, out.SecretScope, "no_permissions_scope")
}
