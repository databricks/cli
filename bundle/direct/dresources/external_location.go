package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
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

func (*ResourceExternalLocation) RemapState(info *catalog.ExternalLocationInfo) *catalog.CreateExternalLocation {
	return &catalog.CreateExternalLocation{
		Comment:                   info.Comment,
		CredentialName:            info.CredentialName,
		EffectiveEnableFileEvents: info.EffectiveEnableFileEvents,
		EnableFileEvents:          info.EnableFileEvents,
		EncryptionDetails:         info.EncryptionDetails,
		Fallback:                  info.Fallback,
		FileEventQueue:            info.FileEventQueue,
		Name:                      info.Name,
		ReadOnly:                  info.ReadOnly,
		SkipValidation:            false, // This is an input-only parameter, never returned by API
		Url:                       info.Url,
		ForceSendFields:           utils.FilterFields[catalog.CreateExternalLocation](info.ForceSendFields),
	}
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

// DoUpdate updates the external location in place and returns remote state.
func (r *ResourceExternalLocation) DoUpdate(ctx context.Context, id string, config *catalog.CreateExternalLocation, _ Changes) (*catalog.ExternalLocationInfo, error) {
	updateRequest := catalog.UpdateExternalLocation{
		Comment:                   config.Comment,
		CredentialName:            config.CredentialName,
		EffectiveEnableFileEvents: config.EffectiveEnableFileEvents,
		EnableFileEvents:          config.EnableFileEvents,
		EncryptionDetails:         config.EncryptionDetails,
		Fallback:                  config.Fallback,
		FileEventQueue:            config.FileEventQueue,
		Force:                     false,
		IsolationMode:             "", // Not supported by DABs
		Name:                      id,
		NewName:                   "", // Only set if name actually changes (see DoUpdateWithID)
		Owner:                     "", // Not supported by DABs
		ReadOnly:                  config.ReadOnly,
		SkipValidation:            config.SkipValidation,
		Url:                       config.Url,
		ForceSendFields:           utils.FilterFields[catalog.UpdateExternalLocation](config.ForceSendFields, "IsolationMode", "Owner"),
	}

	return r.client.ExternalLocations.Update(ctx, updateRequest)
}

// DoUpdateWithID updates the external location and returns the new ID if the name changes.
func (r *ResourceExternalLocation) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateExternalLocation) (string, *catalog.ExternalLocationInfo, error) {
	updateRequest := catalog.UpdateExternalLocation{
		Comment:                   config.Comment,
		CredentialName:            config.CredentialName,
		EffectiveEnableFileEvents: config.EffectiveEnableFileEvents,
		EnableFileEvents:          config.EnableFileEvents,
		EncryptionDetails:         config.EncryptionDetails,
		Fallback:                  config.Fallback,
		FileEventQueue:            config.FileEventQueue,
		Force:                     false,
		IsolationMode:             "", // Not supported by DABs
		Name:                      id,
		NewName:                   "", // Initialized below if needed
		Owner:                     "", // Not supported by DABs
		ReadOnly:                  config.ReadOnly,
		SkipValidation:            config.SkipValidation,
		Url:                       config.Url,
		ForceSendFields:           utils.FilterFields[catalog.UpdateExternalLocation](config.ForceSendFields, "IsolationMode", "Owner"),
	}

	if config.Name != id {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.ExternalLocations.Update(ctx, updateRequest)
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
