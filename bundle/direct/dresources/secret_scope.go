package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type ResourceSecretScope struct {
	client *databricks.WorkspaceClient
}

func (*ResourceSecretScope) New(client *databricks.WorkspaceClient) *ResourceSecretScope {
	return &ResourceSecretScope{client: client}
}

// SecretScopeState represents the state we store for secret scopes
type SecretScopeState struct {
	Name             string                                      `json:"name"`
	BackendType      workspace.ScopeBackendType                  `json:"backend_type,omitempty"`
	KeyvaultMetadata *workspace.AzureKeyVaultSecretScopeMetadata `json:"keyvault_metadata,omitempty"`
}

func (*ResourceSecretScope) PrepareState(input *resources.SecretScope) *SecretScopeState {
	return &SecretScopeState{
		Name:             input.Name,
		BackendType:      input.BackendType,
		KeyvaultMetadata: input.KeyvaultMetadata,
	}
}

func (*ResourceSecretScope) RemapState(info *workspace.SecretScope) *SecretScopeState {
	return &SecretScopeState{
		Name:             info.Name,
		BackendType:      info.BackendType,
		KeyvaultMetadata: info.KeyvaultMetadata,
	}
}

func (r *ResourceSecretScope) DoRead(ctx context.Context, id string) (*workspace.SecretScope, error) {
	// ID is the scope name
	// The Secrets API doesn't have a direct "get by name" endpoint, so we list and find
	scopes, err := r.client.Secrets.ListScopesAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		if scope.Name == id {
			return &scope, nil
		}
	}

	// Return not found error (will be handled as resource gone)
	return nil, &apierr.APIError{
		StatusCode:      404,
		Message:         "Secret scope not found: " + id,
		ErrorCode:       "",
		ResponseWrapper: nil,
		Details:         nil,
	}
}

func (r *ResourceSecretScope) DoCreate(ctx context.Context, config *SecretScopeState) (string, *workspace.SecretScope, error) {
	createRequest := workspace.CreateScope{
		Scope:                  config.Name,
		ScopeBackendType:       config.BackendType,
		BackendAzureKeyvault:   config.KeyvaultMetadata,
		InitialManagePrincipal: "", // Not supported by DABs
		ForceSendFields:        nil,
	}

	err := r.client.Secrets.CreateScope(ctx, createRequest)
	if err != nil {
		return "", nil, err
	}

	// Read back the scope to get remote state
	remoteState, err := r.DoRead(ctx, config.Name)
	if err != nil {
		return config.Name, nil, err
	}

	return config.Name, remoteState, nil
}

// DoUpdate is not applicable for secret scopes - most fields cannot be updated
// The scope would need to be recreated
func (r *ResourceSecretScope) DoUpdate(ctx context.Context, id string, config *SecretScopeState) (*workspace.SecretScope, error) {
	// Secret scopes cannot be updated in place
	// Any changes should trigger a recreate
	// But we still need this method for the interface
	// Just return the current state
	return r.DoRead(ctx, id)
}

func (r *ResourceSecretScope) DoDelete(ctx context.Context, id string) error {
	return r.client.Secrets.DeleteScopeByScope(ctx, id)
}

func (*ResourceSecretScope) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"name":              deployplan.ActionTypeRecreate,
		"backend_type":      deployplan.ActionTypeRecreate,
		"keyvault_metadata": deployplan.ActionTypeRecreate,
	}
}
