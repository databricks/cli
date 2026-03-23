package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceExternalLocation struct {
	client *databricks.WorkspaceClient
}

func (*ResourceExternalLocation) New(client *databricks.WorkspaceClient) *ResourceExternalLocation {
	return &ResourceExternalLocation{client: client}
}

func (*ResourceExternalLocation) PrepareState(input *resources.ExternalLocation) *catalog.CreateExternalLocation {
	return &input.CreateExternalLocation
}

// externalLocationRemapCopy maps ExternalLocationInfo (remote GET response) to CreateExternalLocation (local state).
var externalLocationRemapCopy = newCopy[catalog.ExternalLocationInfo, catalog.CreateExternalLocation]()

func (*ResourceExternalLocation) RemapState(info *catalog.ExternalLocationInfo) *catalog.CreateExternalLocation {
	return externalLocationRemapCopy.Do(info)
}

func (r *ResourceExternalLocation) DoRead(ctx context.Context, id string) (*catalog.ExternalLocationInfo, error) {
	return r.client.ExternalLocations.GetByName(ctx, id)
}

func (r *ResourceExternalLocation) DoCreate(ctx context.Context, config *catalog.CreateExternalLocation) (string, *catalog.ExternalLocationInfo, error) {
	response, err := r.client.ExternalLocations.Create(ctx, *config)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

// externalLocationUpdateCopy maps CreateExternalLocation (local state) to UpdateExternalLocation (API request).
var externalLocationUpdateCopy = newCopy[catalog.CreateExternalLocation, catalog.UpdateExternalLocation]()

// DoUpdate updates the external location in place and returns remote state.
func (r *ResourceExternalLocation) DoUpdate(ctx context.Context, id string, config *catalog.CreateExternalLocation, _ Changes) (*catalog.ExternalLocationInfo, error) {
	updateRequest := externalLocationUpdateCopy.Do(config)
	updateRequest.Name = id

	return r.client.ExternalLocations.Update(ctx, *updateRequest)
}

// DoUpdateWithID updates the external location and returns the new ID if the name changes.
func (r *ResourceExternalLocation) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateExternalLocation) (string, *catalog.ExternalLocationInfo, error) {
	updateRequest := externalLocationUpdateCopy.Do(config)
	updateRequest.Name = id

	if config.Name != id {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.ExternalLocations.Update(ctx, *updateRequest)
	if err != nil {
		return "", nil, err
	}

	// Return the new name as the ID if it changed, otherwise return the old ID
	newID := id
	if updateRequest.NewName != "" {
		newID = updateRequest.NewName
	}

	return newID, response, nil
}

func (r *ResourceExternalLocation) DoDelete(ctx context.Context, id string) error {
	return r.client.ExternalLocations.Delete(ctx, catalog.DeleteExternalLocationRequest{
		Name:            id,
		Force:           true,
		ForceSendFields: nil,
	})
}
