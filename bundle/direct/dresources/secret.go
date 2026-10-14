package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	secrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdksecrets "github.com/databricks/sdk-go/uc/secrets/v1"
)

// Terraform provider implementation:
// https://github.com/databricks/terraform-provider-databricks/blob/main/catalog/resource_secret.go
type ResourceSecret struct {
	client *sdkclients.Clients
}

func (*ResourceSecret) New(client *sdkclients.Clients) *ResourceSecret {
	return &ResourceSecret{client: client}
}

func (*ResourceSecret) PrepareState(input *resources.Secret) *secrets.Secret {
	return &secrets.Secret{
		CatalogName:    input.CatalogName,
		SchemaName:     input.SchemaName,
		Name:           input.Name,
		Value:          input.Value,
		Comment:        input.Comment,
		ExpireTime:     input.ExpireTime,
		Owner:          nil,
		CreateTime:     nil,
		CreatedBy:      nil,
		EffectiveOwner: nil,
		EffectiveValue: nil,
		FullName:       nil,
		MetastoreId:    nil,
		UpdateTime:     nil,
		UpdatedBy:      nil,
	}
}

// remapSecretRemote surfaces the resolved value and owner (the API returns them under
// effective_value/effective_owner) under value/owner so remote state matches the config
// shape — without this, the persisted value is empty and every plan re-reports an update.
// The rest of the secret, including the output-only metadata, is left intact and handled
// by ignore_remote_changes, so no information is discarded. It runs wherever remote state
// is produced (DoRead/DoCreate/DoUpdate), so no RemapState hook is needed.
func remapSecretRemote(resp *sdksecrets.Secret) (*secrets.Secret, error) {
	remote, err := secrets.SecretFromSDK(resp)
	if err != nil {
		return nil, err
	}
	remote.Value = remote.EffectiveValue
	remote.Owner = remote.EffectiveOwner
	return remote, nil
}

// DoRead fetches the secret by full name. IncludeValue is set so remapSecretRemote can
// recover the stored value from EffectiveValue.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*secrets.Secret, error) {
	remote, err := r.client.SecretsUc.GetSecret(ctx, sdksecrets.GetSecretRequest{
		FullName:     new(id),
		IncludeValue: new(true),
	})
	if err != nil {
		return nil, err
	}
	return remapSecretRemote(remote)
}

// DoCreate creates a new UC secret.
func (r *ResourceSecret) DoCreate(ctx context.Context, state *secrets.Secret) (string, *secrets.Secret, error) {
	request, err := secrets.CreateSecretRequestToSDK(&secrets.CreateSecretRequest{Secret: state})
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.SecretsUc.CreateSecret(ctx, *request)
	if err != nil || response == nil {
		return "", nil, err
	}
	remote, err := remapSecretRemote(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(remote.FullName), remote, nil
}

// DoUpdate updates the secret in place and returns remote state.
//
// comment is always sent on update so that clearing it in config clears it on the secret. The
// update_mask is "*" and the backend merges, so an omitted comment would be dropped from the
// body and the old value would drift back. Verified against a real workspace: {"comment": ""} clears it.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *secrets.Secret, _ *PlanEntry) (*secrets.Secret, error) {
	secret := *state
	secret.Comment = orZero(state.Comment)
	request, err := secrets.UpdateSecretRequestToSDK(&secrets.UpdateSecretRequest{
		FullName:   new(id),
		Secret:     &secret,
		UpdateMask: new("*"),
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.SecretsUc.UpdateSecret(ctx, *request)
	if err != nil {
		return nil, err
	}
	return remapSecretRemote(response)
}

// DoDelete deletes the secret.
func (r *ResourceSecret) DoDelete(ctx context.Context, id string, _ *secrets.Secret) error {
	return r.client.SecretsUc.DeleteSecret(ctx, sdksecrets.DeleteSecretRequest{
		FullName: new(id),
	})
}

// OverrideChangeDesc handles the "value" field, which is write-only (the API never
// returns it in GET responses — only effective_value is readable). The state file
// stores "" for this field (never the plaintext), so old is always "" regardless of
// the actual stored value. We compare new vs remote (via effective_value from DoRead)
// to decide whether the secret actually changed: if they are equal, the user's config
// already matches what is stored remotely and no update is needed.
func (*ResourceSecret) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, ch *ChangeDesc, _ *secrets.Secret) error {
	if path.String() != "value" {
		return nil
	}
	if structdiff.IsEqual(ch.Remote, ch.New) {
		ch.Action = deployplan.Skip
		ch.Reason = deployplan.ReasonCustom
	} else {
		ch.Action = deployplan.Update
	}
	return nil
}
