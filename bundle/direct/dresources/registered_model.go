package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
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

// registeredModelRemapCopy maps RegisteredModelInfo (remote GET response) to CreateRegisteredModelRequest (local state).
var registeredModelRemapCopy = newCopy[catalog.RegisteredModelInfo, catalog.CreateRegisteredModelRequest]()

func (*ResourceRegisteredModel) RemapState(model *catalog.RegisteredModelInfo) *catalog.CreateRegisteredModelRequest {
	return registeredModelRemapCopy.Do(model)
}

func (r *ResourceRegisteredModel) DoRead(ctx context.Context, id string) (*catalog.RegisteredModelInfo, error) {
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

// registeredModelUpdateCopy maps CreateRegisteredModelRequest (local state) to UpdateRegisteredModelRequest (API request).
var registeredModelUpdateCopy = newCopy[catalog.CreateRegisteredModelRequest, catalog.UpdateRegisteredModelRequest]()

func (r *ResourceRegisteredModel) DoUpdate(ctx context.Context, id string, config *catalog.CreateRegisteredModelRequest, _ Changes) (*catalog.RegisteredModelInfo, error) {
	updateRequest := registeredModelUpdateCopy.Do(config)
	updateRequest.FullName = id

	response, err := r.client.RegisteredModels.Update(ctx, *updateRequest)
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

