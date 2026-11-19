package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structcopy"
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

// The output-only effective_* fields are mirrored into state to avoid churn in remapped config.
var toCreateExternalLocation = newCopier[catalog.ExternalLocationInfo, catalog.CreateExternalLocation]()

func (*ResourceExternalLocation) RemapState(info *catalog.ExternalLocationInfo) *catalog.CreateExternalLocation {
	return toCreateExternalLocation.Copy(info)
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

// See schemaForceSend. Unlike the other four this one is not probed: an external
// location needs a storage credential with cloud IAM setup, which the test workspaces do
// not provision, so its acceptance test is local-only. Comment is carried here because it
// is the same field on the same UC PATCH family, where all four siblings clear on
// {"comment": ""}; re-probe before adding anything else.
var externalLocationForceSend = []string{"Comment"}

// The effective_* fields are output-only, so they are never sent in an update. Name addresses
// the location, so it is the id; the name in the config goes in NewName when it differs.
var toUpdateExternalLocation = newCopier[catalog.CreateExternalLocation, catalog.UpdateExternalLocation](
	structcopy.Skip("EffectiveEnableFileEvents", "EffectiveFileEventQueue", "Name"),
	structcopy.ForceSend(externalLocationForceSend...),
)

// DoUpdate updates the external location in place and returns remote state.
func (r *ResourceExternalLocation) DoUpdate(ctx context.Context, id string, config *catalog.CreateExternalLocation, _ *PlanEntry) (*catalog.ExternalLocationInfo, error) {
	updateRequest := toUpdateExternalLocation.Copy(config)
	updateRequest.Name = id

	return r.client.ExternalLocations.Update(ctx, *updateRequest)
}

// DoUpdateWithID updates the external location and returns the new ID if the name changes.
func (r *ResourceExternalLocation) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateExternalLocation, _ *PlanEntry) (string, *catalog.ExternalLocationInfo, error) {
	updateRequest := toUpdateExternalLocation.Copy(config)
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

func (r *ResourceExternalLocation) DoDelete(ctx context.Context, id string, _ *catalog.CreateExternalLocation) error {
	return r.client.ExternalLocations.Delete(ctx, catalog.DeleteExternalLocationRequest{
		Name:            id,
		Force:           true,
		ForceSendFields: nil,
	})
}
