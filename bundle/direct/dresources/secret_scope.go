package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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

func (*ResourceSecretScope) RemapState(remote *workspace.SecretScope) *SecretScopeConfig {
	return &SecretScopeConfig{
		CreateScope: workspace.CreateScope{
			Scope:                  remote.Name,
			ScopeBackendType:       remote.BackendType,
			BackendAzureKeyvault:   remote.KeyvaultMetadata,
			InitialManagePrincipal: "",
			ForceSendFields:        utils.FilterFields[workspace.CreateScope](remote.ForceSendFields),
		},
	}
}

// DoRead fetches the secret scope by name. Since the Secrets API does not provide
// a "get by name" endpoint (see https://docs.databricks.com/api/workspace/secrets),
// we must list all scopes and filter by name to check if the scope still exists.
func (r *ResourceSecretScope) DoRead(ctx context.Context, id string) (*workspace.SecretScope, error) {
	scopes, err := r.client.Secrets.ListScopesAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		if scope.Name == id {
			return &scope, nil
		}
	}

	return nil, fmt.Errorf("secret scope %q not found", id)
}

func (r *ResourceSecretScope) DoCreate(ctx context.Context, state *SecretScopeConfig) (string, *workspace.SecretScope, error) {
	err := r.client.Secrets.CreateScope(ctx, state.CreateScope)
	if err != nil {
		return "", nil, err
	}

	return state.Scope, nil, nil
}

// DoUpdate is not intentionally implemented here because scopes do not support a update API. All fields are marked to
// return a recreate trigger.

func (r *ResourceSecretScope) DoDelete(ctx context.Context, id string) error {
	return r.client.Secrets.DeleteScopeByScope(ctx, id)
}

func (r *ResourceSecretScope) FieldTriggers() map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"scope":                    deployplan.ActionTypeRecreate,
		"scope_backend_type":       deployplan.ActionTypeRecreate,
		"backend_azure_keyvault":   deployplan.ActionTypeRecreate,
		"initial_manage_principal": deployplan.ActionTypeRecreate,
	}
}
