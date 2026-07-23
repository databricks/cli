package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
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
	var expireTime *sdktime.Time
	if input.ExpireTime != nil {
		expireTime = sdktime.New(*input.ExpireTime)
	}
	return &catalog.Secret{
		CatalogName:     input.CatalogName,
		SchemaName:      input.SchemaName,
		Name:            input.Name,
		Value:           input.Value,
		Comment:         input.Comment,
		Owner:           input.Owner,
		ExpireTime:      expireTime,
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
		Owner:           remote.Owner,
		ExpireTime:      remote.ExpireTime,
		CreateTime:      nil,
		CreatedBy:       "",
		EffectiveOwner:  "",
		EffectiveValue:  "",
		FullName:        "",
		MetastoreId:     "",
		UpdateTime:      nil,
		UpdatedBy:       "",
		ForceSendFields: utils.FilterFields[catalog.Secret](remote.ForceSendFields),
	}
}

// DoRead fetches the secret by full name.
func (r *ResourceSecret) DoRead(ctx context.Context, id string) (*catalog.Secret, error) {
	return r.client.SecretsUc.GetSecret(ctx, catalog.GetSecretRequest{
		FullName: id,
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
	state.Value = ""
	return response.FullName, response, nil
}

// DoUpdate updates the secret in place and returns remote state.
func (r *ResourceSecret) DoUpdate(ctx context.Context, id string, state *catalog.Secret, _ *PlanEntry) (*catalog.Secret, error) {
	// The UpdateSecret API accepts a field mask specifying which fields to update.
	// Supported fields: value, comment, owner, expire_time
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
	state.Value = ""
	return response, nil
}

// DoDelete deletes the secret.
func (r *ResourceSecret) DoDelete(ctx context.Context, id string, _ *catalog.Secret) error {
	return r.client.SecretsUc.DeleteSecret(ctx, catalog.DeleteSecretRequest{
		FullName: id,
	})
}
