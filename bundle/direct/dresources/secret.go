package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type ResourceSecret struct {
	client *databricks.WorkspaceClient
}

func (*ResourceSecret) New(client *databricks.WorkspaceClient) *ResourceSecret {
	return &ResourceSecret{client: client}
}

// SecretState represents the state we store for secrets
type SecretState struct {
	Scope       string `json:"scope"`
	Key         string `json:"key"`
	StringValue string `json:"string_value,omitempty"`
	BytesValue  string `json:"bytes_value,omitempty"`
}

func (*ResourceSecret) PrepareState(input *resources.Secret) *SecretState {
	return &SecretState{
		Scope:       input.Scope,
		Key:         input.Key,
		StringValue: input.StringValue,
		BytesValue:  input.BytesValue,
	}
}

func (*ResourceSecret) RemapState(info *SecretMetadata) *SecretState {
	// Note: The API doesn't return the actual secret value for security reasons
	// We only track the scope and key in remote state
	return &SecretState{
		Scope:       info.Scope,
		Key:         info.Key,
		StringValue: "", // Not returned by the API for security
		BytesValue:  "", // Not returned by the API for security
	}
}

func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*SecretMetadata, error) {
	// ID format is "scope/key"
	scope, key, err := parseSecretID(id)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Secrets.GetSecret(ctx, workspace.GetSecretRequest{
		Scope: scope,
		Key:   key,
	})
	if err != nil {
		return nil, err
	}

	// Convert to our metadata type that includes scope
	return &SecretMetadata{
		Scope: scope,
		Key:   resp.Key,
	}, nil
}

// SecretMetadata holds metadata about a secret (not the actual value)
type SecretMetadata struct {
	Scope string `json:"scope"`
	Key   string `json:"key"`
}

func (r *ResourceSecret) DoCreate(ctx context.Context, config *SecretState) (string, *SecretMetadata, error) {
	err := r.client.Secrets.PutSecret(ctx, workspace.PutSecret{
		Scope:           config.Scope,
		Key:             config.Key,
		StringValue:     config.StringValue,
		BytesValue:      config.BytesValue,
		ForceSendFields: nil,
	})
	if err != nil {
		return "", nil, err
	}

	id := fmt.Sprintf("%s/%s", config.Scope, config.Key)

	// Read back the secret to get remote state
	remoteState, err := r.DoRead(ctx, id)
	if err != nil {
		return id, nil, err
	}

	return id, remoteState, nil
}

// DoUpdate updates the secret value in place
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, config *SecretState) (*SecretMetadata, error) {
	// PutSecret with the same scope/key updates the value
	err := r.client.Secrets.PutSecret(ctx, workspace.PutSecret{
		Scope:           config.Scope,
		Key:             config.Key,
		StringValue:     config.StringValue,
		BytesValue:      config.BytesValue,
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}

	// Read back the secret to get remote state
	return r.DoRead(ctx, id)
}

func (r *ResourceSecret) DoDelete(ctx context.Context, id string) error {
	scope, key, err := parseSecretID(id)
	if err != nil {
		return err
	}

	return r.client.Secrets.DeleteSecret(ctx, workspace.DeleteSecret{
		Scope: scope,
		Key:   key,
	})
}

func (*ResourceSecret) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"scope": deployplan.ActionTypeRecreate,
		"key":   deployplan.ActionTypeRecreate,
	}
}

// parseSecretID parses a secret ID in the format "scope/key"
func parseSecretID(id string) (string, string, error) {
	// Simple parsing - find the first slash
	for i := range len(id) {
		if id[i] == '/' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid secret ID format: %s (expected scope/key)", id)
}
