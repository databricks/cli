package dresources

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

func (*ResourceMlflowModel) RemapState(model *ml.ModelDatabricks) (*ml.CreateModelRequest, error) {
	return &ml.CreateModelRequest{
		Name:            model.Name,
		Tags:            model.Tags,
		Description:     model.Description,
		ForceSendFields: filterFields[ml.CreateModelRequest](model.ForceSendFields),
	}, nil
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
	// Create API call returns [ml.Model] while DoRefresh returns [ml.ModelDatabricks].
	// Thus we need to convert the response to the expected type.
	modelDatabricks := &ml.ModelDatabricks{
		Name:            response.RegisteredModel.Name,
		Description:     response.RegisteredModel.Description,
		Tags:            response.RegisteredModel.Tags,
		ForceSendFields: filterFields[ml.ModelDatabricks](response.RegisteredModel.ForceSendFields, "CreationTimestamp", "Id", "LastUpdatedTimestamp", "LatestVersions", "PermissionLevel", "UserId"),

		// Coping the fields only to satisfy the linter. These fields are not
		// part of the configuration tree so they don't need to be copied.
		// The linter works as a safeguard to ensure we add new fields to the bundle config tree
		// to the mapping logic here as well.
		CreationTimestamp:    0,
		Id:                   "",
		LastUpdatedTimestamp: 0,
		LatestVersions:       nil,
		PermissionLevel:      "",
		UserId:               "",
	}
	return response.RegisteredModel.Name, modelDatabricks, nil
}

func (r *ResourceMlflowModel) DoUpdate(ctx context.Context, id string, config *ml.CreateModelRequest) (*ml.ModelDatabricks, error) {
	updateRequest := ml.UpdateModelRequest{
		Name:            id,
		Description:     config.Description,
		ForceSendFields: filterFields[ml.UpdateModelRequest](config.ForceSendFields),
	}

	response, err := r.client.ModelRegistry.UpdateModel(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	// Update API call returns [ml.Model] while DoRefresh returns [ml.ModelDatabricks].
	// Thus we need to convert the response to the expected type.
	modelDatabricks := &ml.ModelDatabricks{
		Name:            response.RegisteredModel.Name,
		Description:     response.RegisteredModel.Description,
		Tags:            response.RegisteredModel.Tags,
		ForceSendFields: filterFields[ml.ModelDatabricks](response.RegisteredModel.ForceSendFields, "CreationTimestamp", "Id", "LastUpdatedTimestamp", "LatestVersions", "PermissionLevel", "UserId"),

		// Coping the fields only to satisfy the linter. These fields are not
		// part of the configuration tree so they don't need to be copied.
		// The linter works as a safeguard to ensure we add new fields to the bundle config tree
		// to the mapping logic here as well.
		CreationTimestamp:    0,
		Id:                   "",
		LastUpdatedTimestamp: 0,
		LatestVersions:       nil,
		PermissionLevel:      "",
		UserId:               "",
	}
	return modelDatabricks, nil
}

func (r *ResourceMlflowModel) DoDelete(ctx context.Context, id string) error {
	return r.client.ModelRegistry.DeleteModel(ctx, ml.DeleteModelRequest{
		Name: id,
	})
}

func (*ResourceMlflowModel) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		// Recreate matches current behavior of Terraform. It is possible to rename without recreate
		// but that would require dynamic select of the method during update since
		// the [ml.RenameModel] needs to be called instead of [ml.UpdateModel].
		//
		// We might reasonably choose to never fix this because this is a legacy resource.
		"name": deployplan.ActionTypeRecreate,

		// Allowing updates for tags requires dynamic selection of the method since
		// tags can only be updated by calling [ml.SetModelTag] or [ml.DeleteModelTag] methods.
		//
		// Skip annotation matches the current behavior of Terraform where tags changes are showed
		// in plan but are just ignored / not applied. Since this is a legacy resource we might
		// reasonably choose to not fix it here as well.
		"tags": deployplan.ActionTypeSkip,
	}
}
