package dresources

import (
	"context"

	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/ucm/config/resources"
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
		EffectiveFileEventQueue:   info.EffectiveFileEventQueue,
		EnableFileEvents:          info.EnableFileEvents,
		EncryptionDetails:         info.EncryptionDetails,
		Fallback:                  info.Fallback,
		FileEventQueue:            info.FileEventQueue,
		Name:                      info.Name,
		ReadOnly:                  info.ReadOnly,
		SkipValidation:            false,
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

func (r *ResourceExternalLocation) DoUpdate(ctx context.Context, id string, config *catalog.CreateExternalLocation, _ *PlanEntry) (*catalog.ExternalLocationInfo, error) {
	updateRequest := catalog.UpdateExternalLocation{
		Comment:           config.Comment,
		CredentialName:    config.CredentialName,
		EnableFileEvents:  config.EnableFileEvents,
		EncryptionDetails: config.EncryptionDetails,
		Fallback:          config.Fallback,
		FileEventQueue:    config.FileEventQueue,
		IsolationMode:     "",
		Name:              id,
		Owner:             "",
		ReadOnly:          config.ReadOnly,
		SkipValidation:    config.SkipValidation,
		Url:               config.Url,
		ForceSendFields:   utils.FilterFields[catalog.UpdateExternalLocation](config.ForceSendFields, "IsolationMode", "Owner"),
	}

	return r.client.ExternalLocations.Update(ctx, updateRequest)
}

func (r *ResourceExternalLocation) DoDelete(ctx context.Context, id string) error {
	return r.client.ExternalLocations.DeleteByName(ctx, id)
}
