package dresources

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerTestConfig("secret_scopes", &resources.SecretScope{
		Name:        "my_secret_scope",
		BackendType: workspace.ScopeBackendTypeAzureKeyvault,
		Permissions: []resources.SecretScopePermission{
			{
				Level:    resources.SecretScopePermissionLevelManage,
				UserName: "user@example.com",
			},
		},
		KeyvaultMetadata: &workspace.AzureKeyVaultSecretScopeMetadata{
			DnsName:    "https://my-keyvault.vault.azure.net/",
			ResourceId: "my-keyvault-resource-id",
		},
	})

	registerKnownMissingInRemoteType("secret_scopes",
		"initial_manage_principal",
	)

	registerKnownMissingInStateType("secret_scopes",
		"backend_type",
		"keyvault_metadata",
		"name",
	)

	registerCreateRecoveryOptOut("secret_scopes", "the read response has no create time")
}
