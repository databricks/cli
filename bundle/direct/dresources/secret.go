package dresources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Terraform provider implementation:
// https://github.com/databricks/terraform-provider-databricks/blob/main/catalog/resource_secret.go

type ResourceSecret struct {
	client *databricks.WorkspaceClient
}

// SecretState is the persisted state type for a UC secret. It extends the SDK
// Secret struct with a Fingerprint field so that value changes can be detected
// across deploys without storing the plaintext value on disk. The Value field
// is always cleared after the API call (see DoCreate/DoUpdate).
type SecretState struct {
	catalog.Secret

	// Fingerprint is the SHA-256 hex digest of Value at the time of the last
	// deploy. It lets the planner detect value changes without the plaintext
	// appearing in the state file.
	Fingerprint string `json:"fingerprint"`

	// SecretValue is the plaintext value of the secret. It is not stored in the state file.
	// It is carried here so DoCreate/DoUpdate can send it to the API.
	SecretValue string `json:"-"`
}

func (*ResourceSecret) New(client *databricks.WorkspaceClient) *ResourceSecret {
	return &ResourceSecret{client: client}
}

func (*ResourceSecret) PrepareState(input *resources.Secret) *SecretState {
	var expireTime *sdktime.Time
	if input.ExpireTime != nil {
		expireTime = sdktime.New(*input.ExpireTime)
	}
	return &SecretState{
		Secret: catalog.Secret{
			CatalogName:     input.CatalogName,
			SchemaName:      input.SchemaName,
			Name:            input.Name,
			Value:           "",
			Comment:         input.Comment,
			ExpireTime:      expireTime,
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
		Fingerprint: fingerprintValue(input.Value),
		// Value is carried here so DoCreate/DoUpdate can send it to the API.
		// It is cleared from state after the API call (see DoCreate/DoUpdate).
		SecretValue: input.Value,
	}
}

func (*ResourceSecret) RemapState(remote *catalog.Secret) *SecretState {
	// The GET response never includes the plaintext value, so the fingerprint
	// cannot be reconstructed from remote state. Returning "" causes the planner
	// to treat fingerprint as a remote-only change (suppressed by
	// ignore_remote_changes), so it never drives a spurious update on its own.
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
			EffectiveValue:  "",
			FullName:        "",
			MetastoreId:     "",
			UpdateTime:      nil,
			UpdatedBy:       "",
			ForceSendFields: utils.FilterFields[catalog.Secret](remote.ForceSendFields),
		},
		Fingerprint: "",
		SecretValue: "",
	}
}

// DoRead fetches the secret by full name.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*catalog.Secret, error) {
	return r.client.SecretsUc.GetSecret(ctx, catalog.GetSecretRequest{
		FullName: id,
	})
}

// DoCreate creates a new UC secret.
func (r *ResourceSecret) DoCreate(ctx context.Context, state *SecretState) (string, *catalog.Secret, error) {
	state.Value = state.SecretValue
	response, err := r.client.SecretsUc.CreateSecret(ctx, catalog.CreateSecretRequest{
		Secret: state.Secret,
	})
	// Clear the plaintext so it is not written to the state file.
	// Fingerprint already captures whether the value changed.
	state.Value = ""
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

// DoUpdate updates the secret in place and returns remote state.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *SecretState, _ *PlanEntry) (*catalog.Secret, error) {
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
	return response, nil
}

// DoDelete deletes the secret.
func (r *ResourceSecret) DoDelete(ctx context.Context, id string, _ *SecretState) error {
	return r.client.SecretsUc.DeleteSecret(ctx, catalog.DeleteSecretRequest{
		FullName: id,
	})
}

// MarshalJSON serializes SecretState as a merged JSON object: the fields from
// catalog.Secret (via its own MarshalJSON) plus "fingerprint". Without this,
// the embedded catalog.Secret.MarshalJSON takes over and drops Fingerprint.
func (s SecretState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// UnmarshalJSON deserializes SecretState, restoring both the embedded
// catalog.Secret fields and Fingerprint.
func (s *SecretState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

// fingerprintValue returns the SHA-256 hex digest of v, prefixed with "sha256:"
// to make the algorithm explicit in the stored state.
func fingerprintValue(v string) string {
	sum := sha256.Sum256([]byte(v))
	return "sha256:" + hex.EncodeToString(sum[:])
}
