package dresources

import (
	"context"

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

func (*ResourceSecret) RemapState(remote *catalog.Secret) *catalog.Secret {
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

// DoRead fetches the secret by full name. IncludeValue is set so RemapState can
// recover the stored value from EffectiveValue.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*catalog.Secret, error) {
	return r.client.SecretsUc.GetSecret(ctx, catalog.GetSecretRequest{
		FullName:        id,
		IncludeValue:    true,
		ForceSendFields: nil,
	})
}

// DoCreate creates a new UC secret.
func (r *ResourceSecret) DoCreate(ctx context.Context, state *catalog.Secret) (string, *catalog.Secret, error) {
	response, err := r.client.SecretsUc.CreateSecret(ctx, catalog.CreateSecretRequest{
		Secret: *state,
	})
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

// DoUpdate updates the secret in place and returns remote state.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *catalog.Secret, _ *PlanEntry) (*catalog.Secret, error) {
	response, err := r.client.SecretsUc.UpdateSecret(ctx, catalog.UpdateSecretRequest{
		FullName: id,
		Secret:   *state,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
		},
	})
	if err != nil {
		return nil, err
	}
	return response, nil
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
func (*ResourceSecret) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, ch *ChangeDesc, _, _ *catalog.Secret) error {
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
