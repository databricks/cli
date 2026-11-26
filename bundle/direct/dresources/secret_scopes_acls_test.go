package dresources

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerTestDeps("secret_scopes.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		err := client.Secrets.CreateScope(ctx, workspace.CreateScope{
			Scope:            "permissions_test_scope",
			ScopeBackendType: workspace.ScopeBackendTypeAzureKeyvault,
			BackendAzureKeyvault: &workspace.AzureKeyVaultSecretScopeMetadata{
				DnsName:    "https://my-keyvault.vault.azure.net/",
				ResourceId: "my-keyvault-resource-id",
			},
		})
		if err != nil {
			return nil, err
		}

		return &SecretScopeAclsState{
			ScopeName: "permissions_test_scope",
			Acls: []workspace.AclItem{
				{
					Principal:  "user@example.com",
					Permission: workspace.AclPermissionManage,
				},
			},
		}, nil
	})
}
