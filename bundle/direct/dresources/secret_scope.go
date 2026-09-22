package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type ResourceSecretScope struct {
	client *databricks.WorkspaceClient
}

type SecretScopeConfig struct {
	workspace.CreateScope
}

// SecretScopeRemote is remote state for a secret scope. The Secrets API returns scopes as
// workspace.SecretScope (name/backend_type/keyvault_metadata); DoRead maps those onto the field
// names the state uses (scope/scope_backend_type/backend_azure_keyvault) so RemapState is a plain
// field copy. name is retained because `${resources.secret_scopes.<key>.name}` references resolve
// against the remote type.
type SecretScopeRemote struct {
	Name                 string                                      `json:"name"`
	Scope                string                                      `json:"scope"`
	ScopeBackendType     workspace.ScopeBackendType                  `json:"scope_backend_type,omitempty"`
	BackendAzureKeyvault *workspace.AzureKeyVaultSecretScopeMetadata `json:"backend_azure_keyvault,omitempty"`
	ForceSendFields      []string                                    `json:"-"`
}

func (*ResourceSecretScope) New(client *databricks.WorkspaceClient) *ResourceSecretScope {
	return &ResourceSecretScope{
		client: client,
	}
}

func (*ResourceSecretScope) PrepareState(input *resources.SecretScope) *SecretScopeConfig {
	return &SecretScopeConfig{
		CreateScope: workspace.CreateScope{
			Scope:                  input.Name,
			ScopeBackendType:       input.BackendType,
			BackendAzureKeyvault:   input.KeyvaultMetadata,
			InitialManagePrincipal: "",
			ForceSendFields:        nil,
		},
	}
}

func (*ResourceSecretScope) RemapState(remote *SecretScopeRemote) *SecretScopeConfig {
	return &SecretScopeConfig{
		CreateScope: workspace.CreateScope{
			Scope:                remote.Scope,
			ScopeBackendType:     remote.ScopeBackendType,
			BackendAzureKeyvault: remote.BackendAzureKeyvault,
			ForceSendFields:      utils.FilterFields[workspace.CreateScope](remote.ForceSendFields),
		},
	}
}

// DoRead fetches the secret scope by name. Since the Secrets API does not provide
// a "get by name" endpoint (see https://docs.databricks.com/api/workspace/secrets),
// we must list all scopes and filter by name to check if the scope still exists.
func (r *ResourceSecretScope) DoRead(ctx context.Context, id string) (*SecretScopeRemote, error) {
	scopes, err := r.client.Secrets.ListScopesAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		if scope.Name == id {
			return &SecretScopeRemote{
				Name:                 scope.Name,
				Scope:                scope.Name,
				ScopeBackendType:     scope.BackendType,
				BackendAzureKeyvault: scope.KeyvaultMetadata,
			}, nil
		}
	}

	return nil, fmt.Errorf("secret scope %q not found", id)
}

func (r *ResourceSecretScope) DoCreate(ctx context.Context, state *SecretScopeConfig) (string, *SecretScopeRemote, error) {
	err := r.client.Secrets.CreateScope(ctx, state.CreateScope)
	if err != nil {
		return "", nil, err
	}

	return state.Scope, nil, nil
}

// DoUpdate is not intentionally implemented here because scopes do not support a update API. All fields are marked to
// return a recreate trigger.

func (r *ResourceSecretScope) DoDelete(ctx context.Context, id string, _ *SecretScopeConfig) error {
	return r.client.Secrets.DeleteScopeByScope(ctx, id)
}
