package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	externallocations "github.com/databricks/cli/libs/sdktypes/uc/externallocations/v1"
	"github.com/databricks/cli/libs/utils"
	sdkexternallocations "github.com/databricks/sdk-go/uc/externallocations/v1"
)

type ResourceExternalLocation struct {
	client *sdkclients.Clients
}

func (*ResourceExternalLocation) New(client *sdkclients.Clients) *ResourceExternalLocation {
	return &ResourceExternalLocation{client: client}
}

func (*ResourceExternalLocation) PrepareState(input *resources.ExternalLocation) *resources.CreateExternalLocation {
	return &input.CreateExternalLocation
}

func (*ResourceExternalLocation) RemapState(info *externallocations.ExternalLocationInfo) *resources.CreateExternalLocation {
	return &resources.CreateExternalLocation{
		Comment:        info.Comment,
		CredentialName: info.CredentialName,
		// Output-only fields mirrored into state to avoid churn in remapped config.
		EffectiveEnableFileEvents: info.EffectiveEnableFileEvents,
		EffectiveFileEventQueue:   info.EffectiveFileEventQueue,
		EnableFileEvents:          info.EnableFileEvents,
		EncryptionDetails:         info.EncryptionDetails,
		Fallback:                  info.Fallback,
		FileEventQueue:            info.FileEventQueue,
		Name:                      info.Name,
		ReadOnly:                  info.ReadOnly,
		SkipValidation:            nil, // This is an input-only parameter, never returned by API
		Url:                       info.Url,
	}
}

func (r *ResourceExternalLocation) DoRead(ctx context.Context, id string) (*externallocations.ExternalLocationInfo, error) {
	info, err := r.client.ExternalLocations.GetExternalLocation(ctx, sdkexternallocations.GetExternalLocationRequest{
		NameArg:       new(id),
		IncludeBrowse: nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	return externallocations.ExternalLocationInfoFromSDK(info)
}

func (r *ResourceExternalLocation) DoCreate(ctx context.Context, config *resources.CreateExternalLocation) (string, *externallocations.ExternalLocationInfo, error) {
	request, err := toSDK[sdkexternallocations.CreateExternalLocationRequest](config.AsRequest())
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.ExternalLocations.CreateExternalLocation(ctx, request)
	if err != nil || response == nil {
		return "", nil, err
	}
	info, err := externallocations.ExternalLocationInfoFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(info.Name), info, nil
}

// Comment is always sent, see ResourceSchema.DoUpdate. Unlike the other four this one is not
// probed: an external location needs a storage credential with cloud IAM setup, which the test
// workspaces do not provision, so its acceptance test is local-only. Comment is carried here
// because it is the same field on the same UC PATCH family, where all four siblings clear on
// {"comment": ""}; re-probe before adding anything else.
//
// Output-only effective_* fields are never sent; isolation_mode and owner are not supported by DABs.
func (r *ResourceExternalLocation) update(ctx context.Context, id string, config *resources.CreateExternalLocation, newName *string) (*externallocations.ExternalLocationInfo, error) {
	request, err := toSDK[sdkexternallocations.UpdateExternalLocationRequest](&externallocations.UpdateExternalLocationRequest{
		NameArg:                   new(id),
		NewName:                   newName,
		Comment:                   orZero(config.Comment),
		CredentialName:            config.CredentialName,
		EnableFileEvents:          config.EnableFileEvents,
		EncryptionDetails:         config.EncryptionDetails,
		Fallback:                  config.Fallback,
		FileEventQueue:            config.FileEventQueue,
		ReadOnly:                  config.ReadOnly,
		SkipValidation:            config.SkipValidation,
		Url:                       config.Url,
		BrowseOnly:                nil, // Not set by DABs
		CreatedAt:                 nil, // Not set by DABs
		CreatedBy:                 nil, // Not set by DABs
		CredentialId:              nil, // Not set by DABs
		EffectiveEnableFileEvents: nil, // Not set by DABs
		EffectiveFileEventQueue:   nil, // Not set by DABs
		Force:                     nil, // Not set by DABs
		IsolationMode:             "",  // Not set by DABs
		MetastoreId:               nil, // Not set by DABs
		Name:                      nil, // Not set by DABs
		Owner:                     nil, // Not set by DABs
		UpdatedAt:                 nil, // Not set by DABs
		UpdatedBy:                 nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.ExternalLocations.UpdateExternalLocation(ctx, request)
	if err != nil {
		return nil, err
	}
	return externallocations.ExternalLocationInfoFromSDK(response)
}

// DoUpdate updates the external location in place and returns remote state.
func (r *ResourceExternalLocation) DoUpdate(ctx context.Context, id string, config *resources.CreateExternalLocation, _ *PlanEntry) (*externallocations.ExternalLocationInfo, error) {
	return r.update(ctx, id, config, nil)
}

// DoUpdateWithID updates the external location and returns the new ID if the name changes.
func (r *ResourceExternalLocation) DoUpdateWithID(ctx context.Context, id string, config *resources.CreateExternalLocation, _ *PlanEntry) (string, *externallocations.ExternalLocationInfo, error) {
	var newName *string
	if name := utils.Deref(config.Name); name != id {
		newName = new(name)
	}
	response, err := r.update(ctx, id, config, newName)
	if err != nil {
		return "", nil, err
	}
	if newName != nil {
		return *newName, response, nil
	}
	return id, response, nil
}

func (r *ResourceExternalLocation) DoDelete(ctx context.Context, id string, _ *resources.CreateExternalLocation) error {
	_, err := r.client.ExternalLocations.DeleteExternalLocation(ctx, sdkexternallocations.DeleteExternalLocationRequest{
		NameArg: new(id),
		Force:   new(true),
	})
	return err
}
