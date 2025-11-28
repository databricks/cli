package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSecretScope(t *testing.T) {
	src := resources.SecretScope{
		Name:        "my_scope",
		BackendType: workspace.ScopeBackendTypeDatabricks,
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "my_scope", vin, out)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"name":         "my_scope",
		"backend_type": "DATABRICKS",
	}, out.SecretScope["my_scope"])
}

func TestConvertSecretScopeWithPermissions(t *testing.T) {
	src := resources.SecretScope{
		Name:        "my_scope",
		BackendType: workspace.ScopeBackendTypeDatabricks,
		Permissions: []resources.SecretScopePermission{
			{
				Level:    "READ",
				UserName: "user@example.com",
			},
			{
				Level:     "MANAGE",
				GroupName: "admins",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "my_scope", vin, out)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"name":         "my_scope",
		"backend_type": "DATABRICKS",
	}, out.SecretScope["my_scope"])

	// Verify ACLs are created
	assert.Len(t, out.SecretAcl, 2)

	acl0 := out.SecretAcl["secret_acl_my_scope_0"]
	require.NotNil(t, acl0)
	acl0Typed := acl0.(*resourceSecretAcl)
	assert.Equal(t, "READ", acl0Typed.Permission)
	assert.Equal(t, "user@example.com", acl0Typed.Principal)
	assert.Equal(t, "my_scope", acl0Typed.Scope)
	assert.Contains(t, acl0Typed.DependsOn, "databricks_secret_scope.my_scope")

	acl1 := out.SecretAcl["secret_acl_my_scope_1"]
	require.NotNil(t, acl1)
	acl1Typed := acl1.(*resourceSecretAcl)
	assert.Equal(t, "MANAGE", acl1Typed.Permission)
	assert.Equal(t, "admins", acl1Typed.Principal)
	assert.Equal(t, "my_scope", acl1Typed.Scope)
}

func TestConvertSecretScopeWithAzureKeyvault(t *testing.T) {
	src := resources.SecretScope{
		Name:        "my_keyvault_scope",
		BackendType: workspace.ScopeBackendTypeAzureKeyvault,
		KeyvaultMetadata: &workspace.AzureKeyVaultSecretScopeMetadata{
			DnsName:    "https://mykeyvault.vault.azure.net/",
			ResourceId: "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/mykeyvault",
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretScopeConverter{}.Convert(ctx, "my_keyvault_scope", vin, out)
	require.NoError(t, err)

	scopeOut := out.SecretScope["my_keyvault_scope"].(map[string]any)
	assert.Equal(t, "my_keyvault_scope", scopeOut["name"])
	assert.Equal(t, "AZURE_KEYVAULT", scopeOut["backend_type"])

	keyvaultMeta := scopeOut["keyvault_metadata"].(map[string]any)
	assert.Equal(t, "https://mykeyvault.vault.azure.net/", keyvaultMeta["dns_name"])
	assert.Equal(t, "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/mykeyvault", keyvaultMeta["resource_id"])
}
