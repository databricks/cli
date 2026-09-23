package dresources

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Terraform provider implementation:
// https://github.com/databricks/terraform-provider-databricks/blob/main/catalog/resource_secret.go
type ResourceSecret struct {
	client *databricks.WorkspaceClient
}

func (*ResourceSecret) New(client *databricks.WorkspaceClient) *ResourceSecret {
	return &ResourceSecret{client: client}
}

func (*ResourceSecret) PrepareState(input *resources.Secret) *catalog.Secret {
	return &catalog.Secret{
		CatalogName:     input.CatalogName,
		SchemaName:      input.SchemaName,
		Name:            input.Name,
		Value:           input.Value,
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
	}
}

// remapSecretRemote maps a secret returned by the API to the config's shape: the resolved
// value and owner come back under effective_*, so we surface them under value/owner, and
// output-only fields are dropped. It runs wherever remote state is produced
// (DoRead/DoCreate/DoUpdate), so no RemapState hook is needed.
func remapSecretRemote(remote *catalog.Secret) *catalog.Secret {
	return &catalog.Secret{
		CatalogName:     remote.CatalogName,
		SchemaName:      remote.SchemaName,
		Name:            remote.Name,
		Value:           remote.EffectiveValue,
		Comment:         remote.Comment,
		ExpireTime:      remote.ExpireTime,
		Owner:           remote.EffectiveOwner,
		CreateTime:      nil,
		CreatedBy:       "",
		EffectiveOwner:  "",
		EffectiveValue:  "",
		FullName:        "",
		MetastoreId:     "",
		UpdateTime:      nil,
		UpdatedBy:       "",
		ForceSendFields: utils.FilterFields[catalog.Secret](nil),
	}
}

// DoRead fetches the secret by full name. IncludeValue is set so remapSecretRemote can
// recover the stored value from EffectiveValue.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*catalog.Secret, error) {
	remote, err := r.client.SecretsUc.GetSecret(ctx, catalog.GetSecretRequest{
		FullName:        id,
		IncludeValue:    true,
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}
	return remapSecretRemote(remote), nil
}

// DoCreate creates a new UC secret.
func (r *ResourceSecret) DoCreate(ctx context.Context, state *catalog.Secret) (string, *catalog.Secret, error) {
	response, err := r.client.SecretsUc.CreateSecret(ctx, catalog.CreateSecretRequest{
		Secret: *state,
	})
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.FullName, remapSecretRemote(response), nil
}

// comment is force-sent on update so that clearing it in config clears it on the secret. The
// update_mask is "*" and the backend merges, so an omitempty comment would be dropped from the
// body and the old value would drift back. Verified against a real workspace: {"comment": ""} clears it.
var secretForceSend = []string{"Comment"}

// DoUpdate updates the secret in place and returns remote state.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *catalog.Secret, _ *PlanEntry) (*catalog.Secret, error) {
	secret := *state
	secret.ForceSendFields = utils.FilterFields[catalog.Secret](append(slices.Clone(secretForceSend), state.ForceSendFields...))
	response, err := r.client.SecretsUc.UpdateSecret(ctx, catalog.UpdateSecretRequest{
		FullName: id,
		Secret:   secret,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
		},
	})
	if err != nil {
		return nil, err
	}
	return remapSecretRemote(response), nil
}

// DoDelete deletes the secret.
func (r *ResourceSecret) DoDelete(ctx context.Context, id string, _ *catalog.Secret) error {
	return r.client.SecretsUc.DeleteSecret(ctx, catalog.DeleteSecretRequest{
		FullName: id,
	})
}

// OverrideChangeDesc handles the "value" field, which is write-only (the API never
// returns it in GET responses — only effective_value is readable). The state file
// stores "" for this field (never the plaintext), so old is always "" regardless of
// the actual stored value. We compare new vs remote (via effective_value from DoRead)
// to decide whether the secret actually changed: if they are equal, the user's config
// already matches what is stored remotely and no update is needed.
func (*ResourceSecret) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, ch *ChangeDesc, _ *catalog.Secret) error {
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
