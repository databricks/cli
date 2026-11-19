package dresources

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
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

// Custom marshalers needed because embedded ml.ModelDatabricks has its own
// MarshalJSON which would otherwise take over and ignore model_id.
func (s *MlflowModelRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MlflowModelRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (*ResourceMlflowModel) New(client *databricks.WorkspaceClient) *ResourceMlflowModel {
	return &ResourceMlflowModel{client: client}
}

func (*ResourceMlflowModel) PrepareState(input *resources.MlflowModel) *ml.CreateModelRequest {
	return &input.CreateModelRequest
}

var (
	toCreateModel = newCopier[MlflowModelRemote, ml.CreateModelRequest]()

	// Name addresses the model, so it is the id.
	toUpdateModel = newCopier[ml.CreateModelRequest, ml.UpdateModelRequest](structcopy.Skip("Name"))

	// ml.Model, the UpdateModel response, carries neither Id nor PermissionLevel, so they
	// stay empty; ModelId is carried forward from the remote state the update started from.
	toMlflowModelRemote = newCopier[ml.Model, MlflowModelRemote]()
)

func (*ResourceMlflowModel) RemapState(output *MlflowModelRemote) *ml.CreateModelRequest {
	return toCreateModel.Copy(output)
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
	if response.RegisteredModel == nil {
		return "", nil, errors.New("CreateModel returned no registered_model")
	}
	// Return nil for refresh output; the engine will call DoRead to populate the full state
	// including the numeric model ID needed for permissions.
	return response.RegisteredModel.Name, nil, nil
}

func (r *ResourceMlflowModel) DoUpdate(ctx context.Context, id string, config *ml.CreateModelRequest, entry *PlanEntry) (*MlflowModelRemote, error) {
	updateRequest := toUpdateModel.Copy(config)
	updateRequest.Name = id

	response, err := r.client.ModelRegistry.UpdateModel(ctx, *updateRequest)
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

	// Every field the response has is taken from it, so references against the
	// post-update remote state resolve to real values until the next plan's DoRead.
	remote := toMlflowModelRemote.Copy(response.RegisteredModel)
	remote.ModelId = modelId
	return remote, nil
}

func (r *ResourceMlflowModel) DoDelete(ctx context.Context, id string, _ *ml.CreateModelRequest) error {
	return r.client.ModelRegistry.DeleteModel(ctx, ml.DeleteModelRequest{
		Name: id,
	})
}
