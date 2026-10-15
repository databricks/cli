package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/utils"
	sdksecrets "github.com/databricks/sdk-go/secrets/v1"
)

type ResourceSecretScope struct {
	client *sdkclients.Clients
}

type SecretScopeConfig struct {
	secrets.CreateScopeRequest
}

// SecretScopeRemote is remote state for a secret scope. The Secrets API returns scopes as
// secrets.SecretScope (name/backend_type/keyvault_metadata); DoRead maps those onto the field
// names the state uses (scope/scope_backend_type/backend_azure_keyvault) so RemapState is a plain
// field copy. name is retained because `${resources.secret_scopes.<key>.name}` references resolve
// against the remote type.
type SecretScopeRemote struct {
	Name                 string                                    `json:"name"`
	Scope                string                                    `json:"scope"`
	ScopeBackendType     secrets.ScopeBackendType                  `json:"scope_backend_type,omitempty"`
	BackendAzureKeyvault *secrets.AzureKeyVaultSecretScopeMetadata `json:"backend_azure_keyvault,omitempty"`
}

func (*ResourceSecretScope) New(client *sdkclients.Clients) *ResourceSecretScope {
	return &ResourceSecretScope{
		client: client,
	}
}

func (*ResourceSecretScope) PrepareState(input *resources.SecretScope) *SecretScopeConfig {
	return &SecretScopeConfig{
		Scope:                  new(input.Name),
		ScopeBackendType:       input.BackendType,
		BackendAzureKeyvault:   input.KeyvaultMetadata,
		InitialManagePrincipal: nil,
	}
}

func (*ResourceSecretScope) RemapState(remote *SecretScopeRemote) *SecretScopeConfig {
	return &SecretScopeConfig{
		Scope:                  new(remote.Scope),
		ScopeBackendType:       remote.ScopeBackendType,
		BackendAzureKeyvault:   remote.BackendAzureKeyvault,
		InitialManagePrincipal: nil,
	}
}

// DoRead fetches the secret scope by name. Since the Secrets API does not provide
// a "get by name" endpoint (see https://docs.databricks.com/api/workspace/secrets),
// we must list all scopes and filter by name to check if the scope still exists.
func (r *ResourceSecretScope) DoRead(ctx context.Context, id string) (*SecretScopeRemote, error) {
	resp, err := r.client.Secrets.ListScopes(ctx, sdksecrets.ListScopesRequest{})
	if err != nil {
		return nil, err
	}
	scopes, err := secrets.ListScopesResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes.Scopes {
		if name := utils.Deref(scope.Name); name == id {
			return &SecretScopeRemote{
				Name:                 name,
				Scope:                name,
				ScopeBackendType:     scope.BackendType,
				BackendAzureKeyvault: scope.KeyvaultMetadata,
			}, nil
		}
	}

	return nil, errResourceDoesNotExist("secret scope " + id + " does not exist")
}

func (r *ResourceSecretScope) DoCreate(ctx context.Context, state *SecretScopeConfig) (string, *SecretScopeRemote, error) {
	request, err := secrets.CreateScopeRequestToSDK(&state.CreateScopeRequest)
	if err != nil {
		return "", nil, err
	}
	if _, err := r.client.Secrets.CreateScope(ctx, *request); err != nil {
		return "", nil, err
	}

	return utils.Deref(state.Scope), nil, nil
}

// DoUpdate is not intentionally implemented here because scopes do not support a update API. All fields are marked to
// return a recreate trigger.

func (r *ResourceSecretScope) DoDelete(ctx context.Context, id string, _ *SecretScopeConfig) error {
	_, err := r.client.Secrets.DeleteScope(ctx, sdksecrets.DeleteScopeRequest{Scope: new(id)})
	return err
}
