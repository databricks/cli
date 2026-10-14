package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	registeredmodels "github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	"github.com/databricks/cli/libs/utils"
	sdkregisteredmodels "github.com/databricks/sdk-go/uc/registeredmodels/v1"
)

type ResourceRegisteredModel struct {
	client *sdkclients.Clients
}

func (*ResourceRegisteredModel) New(client *sdkclients.Clients) *ResourceRegisteredModel {
	return &ResourceRegisteredModel{
		client: client,
	}
}

func (*ResourceRegisteredModel) PrepareState(input *resources.RegisteredModel) *registeredmodels.CreateRegisteredModelRequest {
	return &input.CreateRegisteredModelRequest
}

func (*ResourceRegisteredModel) RemapState(model *registeredmodels.RegisteredModelInfo) *registeredmodels.CreateRegisteredModelRequest {
	return &registeredmodels.CreateRegisteredModelRequest{
		CatalogName:     model.CatalogName,
		Comment:         model.Comment,
		Name:            model.Name,
		SchemaName:      model.SchemaName,
		StorageLocation: model.StorageLocation,

		Aliases:     model.Aliases,
		BrowseOnly:  model.BrowseOnly,
		FullName:    model.FullName,
		MetastoreId: model.MetastoreId,
		Owner:       model.Owner,

		// Output only fields. Remote changes to these are ignored via
		// ignore_remote_changes in registered_models.yml rather than zeroed here.
		CreatedAt: model.CreatedAt,
		CreatedBy: model.CreatedBy,
		UpdatedAt: model.UpdatedAt,
		UpdatedBy: model.UpdatedBy,
	}
}

func (r *ResourceRegisteredModel) DoRead(ctx context.Context, id string) (*registeredmodels.RegisteredModelInfo, error) {
	info, err := r.client.RegisteredModels.GetRegisteredModel(ctx, sdkregisteredmodels.GetRegisteredModelRequest{
		FullNameArg: new(id),
	})
	if err != nil {
		return nil, err
	}
	return registeredmodels.RegisteredModelInfoFromSDK(info)
}

func (r *ResourceRegisteredModel) DoCreate(ctx context.Context, config *registeredmodels.CreateRegisteredModelRequest) (string, *registeredmodels.RegisteredModelInfo, error) {
	request, err := registeredmodels.CreateRegisteredModelRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.RegisteredModels.CreateRegisteredModel(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	info, err := registeredmodels.RegisteredModelInfoFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(info.FullName), info, nil
}

func (r *ResourceRegisteredModel) DoUpdate(ctx context.Context, id string, config *registeredmodels.CreateRegisteredModelRequest, _ *PlanEntry) (*registeredmodels.RegisteredModelInfo, error) {
	request, err := registeredmodels.UpdateRegisteredModelRequestToSDK(&registeredmodels.UpdateRegisteredModelRequest{
		FullNameArg: new(id),
		// Always sent, see ResourceSchema.DoUpdate. Verified against a real workspace: {"comment": ""} clears it.
		Comment: orZero(config.Comment),

		// Owner is settable in the config (it comes from the embedded
		// CreateRegisteredModelRequest) and create sends it, but update never has: a
		// change to it is silently dropped rather than applied.
		Owner: nil,

		// Name updates are not supported yet without recreating. Can be added as a follow-up.
		// Note: TF also does not support changing name without a recreate so the current behavior matches TF.
		NewName: nil,

		Aliases:         config.Aliases,
		BrowseOnly:      config.BrowseOnly,
		CreatedAt:       config.CreatedAt,
		CreatedBy:       config.CreatedBy,
		MetastoreId:     config.MetastoreId,
		UpdatedAt:       config.UpdatedAt,
		UpdatedBy:       config.UpdatedBy,
		SchemaName:      config.SchemaName,
		StorageLocation: config.StorageLocation,
		Name:            config.Name,
		CatalogName:     config.CatalogName,
		FullName:        nil,
	})
	if err != nil {
		return nil, err
	}

	response, err := r.client.RegisteredModels.UpdateRegisteredModel(ctx, *request)
	if err != nil {
		return nil, err
	}
	return registeredmodels.RegisteredModelInfoFromSDK(response)
}

func (r *ResourceRegisteredModel) DoDelete(ctx context.Context, id string, _ *registeredmodels.CreateRegisteredModelRequest) error {
	_, err := r.client.RegisteredModels.DeleteRegisteredModel(ctx, sdkregisteredmodels.DeleteRegisteredModelRequest{
		FullNameArg: new(id),
	})
	return err
}
