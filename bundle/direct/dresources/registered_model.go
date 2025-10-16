package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceRegisteredModel struct {
	client *databricks.WorkspaceClient
}

func (*ResourceRegisteredModel) New(client *databricks.WorkspaceClient) *ResourceRegisteredModel {
	return &ResourceRegisteredModel{
		client: client,
	}
}

func (*ResourceRegisteredModel) PrepareState(input *resources.RegisteredModel) *catalog.CreateRegisteredModelRequest {
	return &input.CreateRegisteredModelRequest
}

func (*ResourceRegisteredModel) RemapState(model *catalog.RegisteredModelInfo) *catalog.CreateRegisteredModelRequest {
	return &catalog.CreateRegisteredModelRequest{
		CatalogName:     model.CatalogName,
		Comment:         model.Comment,
		Name:            model.Name,
		SchemaName:      model.SchemaName,
		StorageLocation: model.StorageLocation,
		ForceSendFields: filterFields[catalog.CreateRegisteredModelRequest](model.ForceSendFields),

		Aliases:     model.Aliases,
		BrowseOnly:  model.BrowseOnly,
		CreatedAt:   model.CreatedAt,
		CreatedBy:   model.CreatedBy,
		FullName:    model.FullName,
		MetastoreId: model.MetastoreId,
		Owner:       model.Owner,
		UpdatedAt:   model.UpdatedAt,
		UpdatedBy:   model.UpdatedBy,
	}
}

func (r *ResourceRegisteredModel) DoRefresh(ctx context.Context, id string) (*catalog.RegisteredModelInfo, error) {
	return r.client.RegisteredModels.Get(ctx, catalog.GetRegisteredModelRequest{
		FullName:        id,
		IncludeAliases:  false,
		IncludeBrowse:   false,
		ForceSendFields: nil,
	})
}

func (r *ResourceRegisteredModel) DoCreate(ctx context.Context, config *catalog.CreateRegisteredModelRequest) (string, *catalog.RegisteredModelInfo, error) {
	response, err := r.client.RegisteredModels.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return response.FullName, response, nil
}

func (r *ResourceRegisteredModel) DoUpdate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest) (*catalog.RegisteredModelInfo, error) {
	updateRequest := catalog.UpdateRegisteredModelRequest{
		FullName:        id,
		Comment:         config.Comment,
		ForceSendFields: filterFields[catalog.UpdateRegisteredModelRequest](config.ForceSendFields, "Owner", "NewName"),

		// Owner is not part of the configuration tree
		Owner: "",

		// Name updates are not supported yet without recreating. Can be added as a follow-up.
		// Note: TF also does not support changing name without a recreate so the current behavior matches TF.
		NewName: "",

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
	}

	response, err := r.client.RegisteredModels.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ResourceRegisteredModel) DoDelete(ctx context.Context, id string) error {
	return r.client.RegisteredModels.Delete(ctx, catalog.DeleteRegisteredModelRequest{
		FullName: id,
	})
}

func (*ResourceRegisteredModel) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		// The name can technically be updated without recreated. We recreate for now though
		// to match TF implementation.
		"name": deployplan.ActionTypeRecreate,

		"catalog_name":     deployplan.ActionTypeRecreate,
		"schema_name":      deployplan.ActionTypeRecreate,
		"storage_location": deployplan.ActionTypeRecreate,
	}
}
