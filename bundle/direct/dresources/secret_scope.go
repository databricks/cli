package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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
			ForceSendFields:        filterFields[workspace.CreateScope](remote.ForceSendFields),
		},
	}
}

func (r *ResourceSecretScope) DoRefresh(ctx context.Context, id string) (*workspace.SecretScope, error) {
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

func (r *ResourceSecretScope) DoCreate(ctx context.Context, state *SecretScopeConfig) (string, error) {
	err := r.client.Secrets.CreateScope(ctx, state.CreateScope)
	if err != nil {
		return "", err
	}

	return state.Scope, nil
}

func (r *ResourceSecretScope) DoUpdate(ctx context.Context, id string, state *SecretScopeConfig) error {
	// Secret scopes themselves are immutable
	return fmt.Errorf("secret scopes cannot be updated, they must be recreated")
}

func (r *ResourceSecretScope) DoDelete(ctx context.Context, id string) error {
	return r.client.Secrets.DeleteScopeByScope(ctx, id)
}

func (r *ResourceSecretScope) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"scope":                    deployplan.ActionTypeRecreate,
		"scope_backend_type":       deployplan.ActionTypeRecreate,
		"backend_azure_keyvault":   deployplan.ActionTypeRecreate,
		"initial_manage_principal": deployplan.ActionTypeRecreate,
	}
}
