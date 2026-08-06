package dresources

import (
	"context"
	"net/http"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Terraform provider implementation:
// https://github.com/databricks/terraform-provider-databricks/blob/main/catalog/resource_secret.go
type ResourceSecret struct {
	client *databricks.WorkspaceClient
}

// SecretRemote is the remote read type for a UC secret. It embeds the SDK Secret
// struct and adds SecretValue, populated from EffectiveValue by DoRead, so that
// drift detection can compare the live secret value against the desired config value.
type SecretRemote struct {
	catalog.Secret

	// SecretValue mirrors EffectiveValue and is populated by DoRead when include_value=true.
	// It uses the same field name as SecretState.SecretValue so RemapState can copy it directly.
	SecretValue string `json:"-" bundle:"sensitive"`
}

// SecretState is the persisted state type for a UC secret. It extends the SDK
// Secret struct with a Fingerprint field so that value changes can be detected
// across deploys without storing the plaintext value on disk. The Value field
// is always cleared after the API call (see DoCreate/DoUpdate).
type SecretState struct {
	catalog.Secret

	// SecretValue is the plaintext value of the secret. It is not stored in the state file.
	// It is carried here so DoCreate/DoUpdate can send it to the API.
	SecretValue string `json:"-" bundle:"sensitive"`
}

func (*ResourceSecret) New(client *databricks.WorkspaceClient) *ResourceSecret {
	return &ResourceSecret{client: client}
}

func (*ResourceSecret) PrepareState(input *resources.Secret) *SecretState {
	return &SecretState{
		Secret: catalog.Secret{
			CatalogName:     input.CatalogName,
			SchemaName:      input.SchemaName,
			Name:            input.Name,
			Value:           "",
			Comment:         input.Comment,
			ExpireTime:      input.ExpireTime,
			Owner:           "",
			CreateTime:      nil,
			CreatedBy:       "",
			EffectiveOwner:  "",
			EffectiveValue:  "",
			FullName:        "",
			MetastoreId:     "",
			UpdateTime:      nil,
			UpdatedBy:       "",
			ForceSendFields: utils.FilterFields[catalog.Secret](nil),
		},
		// Value is carried here so DoCreate/DoUpdate can send it to the API.
		// It is cleared from state after the API call (see DoCreate/DoUpdate).
		SecretValue: input.Value,
	}
}

func (*ResourceSecret) RemapState(remote *SecretRemote) *SecretState {
	return &SecretState{
		Secret: catalog.Secret{
			CatalogName:     remote.CatalogName,
			SchemaName:      remote.SchemaName,
			Name:            remote.Name,
			Comment:         remote.Comment,
			Owner:           remote.Owner,
			ExpireTime:      remote.ExpireTime,
			Value:           "",
			CreateTime:      nil,
			CreatedBy:       "",
			EffectiveOwner:  "",
			EffectiveValue:  remote.EffectiveValue,
			FullName:        "",
			MetastoreId:     "",
			UpdateTime:      nil,
			UpdatedBy:       "",
			ForceSendFields: utils.FilterFields[catalog.Secret](remote.ForceSendFields),
		},
		SecretValue: remote.SecretValue,
	}
}

// DoRead fetches the secret by full name.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*SecretRemote, error) {
	apiClient, err := client.New(r.client.Config)
	if err != nil {
		return nil, err
	}

	// SDK does not support include_value in the GetSecretRequest, so we use the API directly.
	var secret catalog.Secret
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.1/unity-catalog/secrets/"+id, auth.WorkspaceIDHeaders(r.client.Config), map[string]any{
		"include_value": true,
	}, nil, &secret)
	if err != nil {
		return nil, err
	}
	// Populate SecretValue from EffectiveValue so drift detection can compare
	// the live secret value against the desired config value via RemapState.
	return &SecretRemote{
		Secret:      secret,
		SecretValue: secret.EffectiveValue,
	}, nil
}

// DoCreate creates a new UC secret.
func (r *ResourceSecret) DoCreate(ctx context.Context, state *SecretState) (string, *SecretRemote, error) {
	state.Value = state.SecretValue
	response, err := r.client.SecretsUc.CreateSecret(ctx, catalog.CreateSecretRequest{
		Secret: state.Secret,
	})
	// Clear the plaintext so it is not written to the state file.
	state.Value = ""
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.FullName, &SecretRemote{Secret: *response, SecretValue: state.SecretValue}, nil
}

// DoUpdate updates the secret in place and returns remote state.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *SecretState, _ *PlanEntry) (*SecretRemote, error) {
	state.Value = state.SecretValue
	response, err := r.client.SecretsUc.UpdateSecret(ctx, catalog.UpdateSecretRequest{
		FullName: id,
		Secret:   state.Secret,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
		},
	})
	// Clear the plaintext so it is not written to the state file.
	state.Value = ""
	if err != nil {
		return nil, err
	}
	return &SecretRemote{Secret: *response, SecretValue: state.SecretValue}, nil
}

// DoDelete deletes the secret.
func (r *ResourceSecret) DoDelete(ctx context.Context, id string, _ *SecretState) error {
	return r.client.SecretsUc.DeleteSecret(ctx, catalog.DeleteSecretRequest{
		FullName: id,
	})
}

// MarshalJSON serializes SecretRemote as a merged JSON object: the fields from
// catalog.Secret (via its own MarshalJSON) plus any SecretRemote-specific fields.
// Without this, the embedded catalog.Secret.MarshalJSON takes over and drops them.
func (s SecretRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// UnmarshalJSON deserializes SecretRemote, restoring both the embedded
// catalog.Secret fields and any SecretRemote-specific fields.
func (s *SecretRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

// MarshalJSON serializes SecretState as a merged JSON object: the fields from
// catalog.Secret (via its own MarshalJSON) plus SecretState-specific fields.
// Without this, the embedded catalog.Secret.MarshalJSON takes over and drops them.
func (s SecretState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// UnmarshalJSON deserializes SecretState, restoring both the embedded
// catalog.Secret fields and SecretState-specific fields.
func (s *SecretState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}
