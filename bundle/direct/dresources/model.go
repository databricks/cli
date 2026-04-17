package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type ResourceMlflowModel struct {
	client *databricks.WorkspaceClient
}

// MlflowModelRemote wraps the API response with the numeric model ID.
// The state ID for models is the model name (used for CRUD operations), but
// the permissions API requires the numeric ID. This wrapper exposes the numeric
// ID as model_id, analogous to ModelServingEndpointRemote.EndpointId for serving endpoints.
type MlflowModelRemote struct {
	ml.ModelDatabricks
	ModelId string `json:"model_id"`
}

func (*ResourceMlflowModel) New(client *databricks.WorkspaceClient) *ResourceMlflowModel {
	return &ResourceMlflowModel{client: client}
}

func (*ResourceMlflowModel) PrepareState(input *resources.MlflowModel) *ml.CreateModelRequest {
	return &input.CreateModelRequest
}

func (*ResourceMlflowModel) RemapState(output *MlflowModelRemote) *ml.CreateModelRequest {
	return &ml.CreateModelRequest{
		Name:            output.Name,
		Tags:            output.Tags,
		Description:     output.Description,
		ForceSendFields: utils.FilterFields[ml.CreateModelRequest](output.ForceSendFields),
	}
}

func (r *ResourceMlflowModel) DoRead(ctx context.Context, id string) (*MlflowModelRemote, error) {
	response, err := r.client.ModelRegistry.GetModel(ctx, ml.GetModelRequest{
		Name: id,
	})
	if err != nil {
		return nil, err
	}
	return &MlflowModelRemote{
		ModelDatabricks: *response.RegisteredModelDatabricks,
		ModelId:         response.RegisteredModelDatabricks.Id,
	}, nil
}

func (r *ResourceMlflowModel) DoCreate(ctx context.Context, config *ml.CreateModelRequest) (string, *MlflowModelRemote, error) {
	response, err := r.client.ModelRegistry.CreateModel(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	// Return nil for refresh output; the engine will call DoRead to populate the full state
	// including the numeric model ID needed for permissions.
	return response.RegisteredModel.Name, nil, nil
}

func (r *ResourceMlflowModel) DoUpdate(ctx context.Context, id string, config *ml.CreateModelRequest, entry *PlanEntry) (*MlflowModelRemote, error) {
	updateRequest := ml.UpdateModelRequest{
		Name:            id,
		Description:     config.Description,
		ForceSendFields: utils.FilterFields[ml.UpdateModelRequest](config.ForceSendFields),
	}

	response, err := r.client.ModelRegistry.UpdateModel(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	// Carry forward model_id from existing state since UpdateModelResponse doesn't include it.
	var modelId string
	if old, ok := entry.RemoteState.(*MlflowModelRemote); ok {
		modelId = old.ModelId
	}

	return &MlflowModelRemote{
		ModelDatabricks: ml.ModelDatabricks{
			CreationTimestamp:    0,
			Description:          response.RegisteredModel.Description,
			Id:                   "",
			LastUpdatedTimestamp: 0,
			LatestVersions:       nil,
			Name:                 response.RegisteredModel.Name,
			PermissionLevel:      "",
			Tags:                 response.RegisteredModel.Tags,
			UserId:               "",
			ForceSendFields:      utils.FilterFields[ml.ModelDatabricks](response.RegisteredModel.ForceSendFields, "CreationTimestamp", "Id", "LastUpdatedTimestamp", "LatestVersions", "PermissionLevel", "UserId"),
		},
		ModelId: modelId,
	}, nil
}

func (r *ResourceMlflowModel) DoDelete(ctx context.Context, id string) error {
	return r.client.ModelRegistry.DeleteModel(ctx, ml.DeleteModelRequest{
		Name: id,
	})
}
