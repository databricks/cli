package dresources

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	modelregistry "github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"github.com/databricks/cli/libs/utils"
	sdkmodelregistry "github.com/databricks/sdk-go/modelregistry/v1"
)

type ResourceMlflowModel struct {
	client *sdkclients.Clients
}

// MlflowModelRemote wraps the API response with the numeric model ID.
// The state ID for models is the model name (used for CRUD operations), but
// the permissions API requires the numeric ID. This wrapper exposes the numeric
// ID as model_id, analogous to ModelServingEndpointRemote.EndpointId for serving endpoints.
type MlflowModelRemote struct {
	modelregistry.RegisteredModelDatabricks
	ModelId string `json:"model_id"`
}

func (*ResourceMlflowModel) New(client *sdkclients.Clients) *ResourceMlflowModel {
	return &ResourceMlflowModel{client: client}
}

func (*ResourceMlflowModel) PrepareState(input *resources.MlflowModel) *modelregistry.CreateRegisteredModelRequest {
	return &input.CreateRegisteredModelRequest
}

func (*ResourceMlflowModel) RemapState(output *MlflowModelRemote) *modelregistry.CreateRegisteredModelRequest {
	return &modelregistry.CreateRegisteredModelRequest{
		Name:        output.Name,
		Tags:        output.Tags,
		Description: output.Description,
	}
}

func (r *ResourceMlflowModel) DoRead(ctx context.Context, id string) (*MlflowModelRemote, error) {
	resp, err := r.client.ModelRegistry.GetRegisteredModelDatabricks(ctx, sdkmodelregistry.GetRegisteredModelDatabricksRequest{
		Name: new(id),
	})
	if err != nil {
		return nil, err
	}
	response, err := modelregistry.GetRegisteredModelDatabricksResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	if response.RegisteredModelDatabricks == nil {
		return nil, errors.New("GetModel returned no registered_model_databricks")
	}
	return &MlflowModelRemote{
		RegisteredModelDatabricks: *response.RegisteredModelDatabricks,
		ModelId:                   utils.Deref(response.RegisteredModelDatabricks.Id),
	}, nil
}

func (r *ResourceMlflowModel) DoCreate(ctx context.Context, config *modelregistry.CreateRegisteredModelRequest) (string, *MlflowModelRemote, error) {
	request, err := modelregistry.CreateRegisteredModelRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.ModelRegistry.CreateRegisteredModel(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	if response.RegisteredModel == nil {
		return "", nil, errors.New("CreateModel returned no registered_model")
	}
	// Return nil for refresh output; the engine will call DoRead to populate the full state
	// including the numeric model ID needed for permissions.
	return utils.Deref(response.RegisteredModel.Name), nil, nil
}

func (r *ResourceMlflowModel) DoUpdate(ctx context.Context, id string, config *modelregistry.CreateRegisteredModelRequest, entry *PlanEntry) (*MlflowModelRemote, error) {
	resp, err := r.client.ModelRegistry.UpdateRegisteredModel(ctx, sdkmodelregistry.UpdateRegisteredModelRequest{
		Name:        new(id),
		Description: config.Description,
	})
	if err != nil {
		return nil, err
	}
	response, err := modelregistry.UpdateRegisteredModelResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	if response.RegisteredModel == nil {
		return nil, errors.New("UpdateModel returned no registered_model")
	}

	// Carry forward model_id from existing state since UpdateModelResponse doesn't include it.
	var modelId string
	if old, ok := entry.RemoteState.(*MlflowModelRemote); ok {
		modelId = old.ModelId
	}

	// Id and PermissionLevel are left empty because RegisteredModel (the update
	// response) does not carry them; every other field is taken from the
	// response so references against the post-update remote state resolve to
	// real values until the next plan's DoRead.
	model := response.RegisteredModel
	return &MlflowModelRemote{
		RegisteredModelDatabricks: modelregistry.RegisteredModelDatabricks{
			CreationTimestamp:    model.CreationTimestamp,
			Description:          model.Description,
			LastUpdatedTimestamp: model.LastUpdatedTimestamp,
			LatestVersions:       model.LatestVersions,
			Name:                 model.Name,
			Tags:                 model.Tags,
			UserId:               model.UserId,
		},
		ModelId: modelId,
	}, nil
}

func (r *ResourceMlflowModel) DoDelete(ctx context.Context, id string, _ *modelregistry.CreateRegisteredModelRequest) error {
	_, err := r.client.ModelRegistry.DeleteRegisteredModel(ctx, sdkmodelregistry.DeleteRegisteredModelRequest{
		Name: new(id),
	})
	return err
}
