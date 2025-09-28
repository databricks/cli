package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type ResourceMlflowModel struct {
	client *databricks.WorkspaceClient
}

func (*ResourceMlflowModel) New(client *databricks.WorkspaceClient) *ResourceMlflowModel {
	return &ResourceMlflowModel{client: client}
}

func (*ResourceMlflowModel) PrepareState(input *resources.MlflowModel) *ml.CreateModelRequest {
	return &input.CreateModelRequest
}

func (*ResourceMlflowModel) RemapState(model *ml.ModelDatabricks) *ml.CreateModelRequest {
	return &ml.CreateModelRequest{
		Name:            model.Name,
		Tags:            model.Tags,
		Description:     model.Description,
		ForceSendFields: filterFields[ml.CreateModelRequest](model.ForceSendFields),
	}
}

func (r *ResourceMlflowModel) DoRefresh(ctx context.Context, id string) (*ml.ModelDatabricks, error) {
	response, err := r.client.ModelRegistry.GetModel(ctx, ml.GetModelRequest{
		Name: id,
	})
	if err != nil {
		return nil, err
	}
	return response.RegisteredModelDatabricks, nil
}

func (r *ResourceMlflowModel) DoCreate(ctx context.Context, config *ml.CreateModelRequest) (string, *ml.ModelDatabricks, error) {
	response, err := r.client.ModelRegistry.CreateModel(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	// Convert ml.Model to ml.ModelDatabricks for consistency with DoRefresh
	modelDatabricks := &ml.ModelDatabricks{
		Name:                 response.RegisteredModel.Name,
		Description:          response.RegisteredModel.Description,
		Tags:                 response.RegisteredModel.Tags,
		CreationTimestamp:    response.RegisteredModel.CreationTimestamp,
		LastUpdatedTimestamp: response.RegisteredModel.LastUpdatedTimestamp,
		UserId:               response.RegisteredModel.UserId,
		// Note: Some fields like LatestVersions might be missing in the conversion
	}
	return response.RegisteredModel.Name, modelDatabricks, nil
}

func (r *ResourceMlflowModel) DoUpdate(ctx context.Context, id string, config *ml.CreateModelRequest) (*ml.ModelDatabricks, error) {
	updateRequest := ml.UpdateModelRequest{
		Name:        id,
		Description: config.Description,
		// Note: Name changes are not supported by the MLflow model registry API
		// Tags are updated separately via SetModelTag/DeleteModelTag operations
		// For simplicity, we only support description updates
	}

	response, err := r.client.ModelRegistry.UpdateModel(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	// Convert ml.Model to ml.ModelDatabricks for consistency with DoRefresh
	modelDatabricks := &ml.ModelDatabricks{
		Name:                 response.RegisteredModel.Name,
		Description:          response.RegisteredModel.Description,
		Tags:                 response.RegisteredModel.Tags,
		CreationTimestamp:    response.RegisteredModel.CreationTimestamp,
		LastUpdatedTimestamp: response.RegisteredModel.LastUpdatedTimestamp,
		UserId:               response.RegisteredModel.UserId,
		// Note: Some fields like LatestVersions might be missing in the conversion
	}
	return modelDatabricks, nil
}

func (r *ResourceMlflowModel) DoDelete(ctx context.Context, id string) error {
	return r.client.ModelRegistry.DeleteModel(ctx, ml.DeleteModelRequest{
		Name: id,
	})
}

func (*ResourceMlflowModel) FieldTriggers() map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		".name": deployplan.ActionTypeRecreate, // Name changes require recreation
		// Description changes can be updated in place
		// Tags are handled separately and don't require special triggers
	}
}
