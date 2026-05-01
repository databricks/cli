package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceCatalog struct {
	client *databricks.WorkspaceClient
}

func (*ResourceCatalog) New(client *databricks.WorkspaceClient) *ResourceCatalog {
	return &ResourceCatalog{client: client}
}

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *catalog.CreateCatalog {
	return &input.CreateCatalog
}

func (*ResourceCatalog) RemapState(info *catalog.CatalogInfo) *catalog.CreateCatalog {
	return &catalog.CreateCatalog{
		Comment:                   info.Comment,
		ConnectionName:            info.ConnectionName,
		ManagedEncryptionSettings: info.ManagedEncryptionSettings,
		Name:                      info.Name,
		Options:                   info.Options,
		Properties:                info.Properties,
		ProviderName:              info.ProviderName,
		ShareName:                 info.ShareName,
		StorageRoot:               info.StorageRoot,
		ForceSendFields:           utils.FilterFields[catalog.CreateCatalog](info.ForceSendFields),
	}
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*catalog.CatalogInfo, error) {
	return r.client.Catalogs.GetByName(ctx, id)
}

func (r *ResourceCatalog) DoCreate(ctx context.Context, config *catalog.CreateCatalog) (string, *catalog.CatalogInfo, error) {
	response, err := r.client.Catalogs.Create(ctx, *config)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *catalog.CreateCatalog, _ *PlanEntry) (*catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                "", // Not supported by DABs
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Only set if name actually changes (see DoUpdateWithID)
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](config.ForceSendFields, "EnablePredictiveOptimization", "IsolationMode", "Owner"),
	}

	response, err := r.client.Catalogs.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateCatalog) (string, *catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                "", // Not supported by DABs
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Initialized below if needed
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](config.ForceSendFields, "EnablePredictiveOptimization", "IsolationMode", "Owner"),
	}

	if config.Name != id {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.Catalogs.Update(ctx, updateRequest)
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

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string) error {
	return r.client.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{
		Name:            id,
		Force:           true,
		ForceSendFields: nil,
	})
}

// OverrideChangeDesc suppresses drift for storage_root when the only difference
// is a trailing slash. The UC API strips trailing slashes on create, so remote
// returns "s3://bucket/path" while the config may have "s3://bucket/path/".
// Without this, storage_root being in recreate_on_changes triggers a destructive
// delete + create on every deploy.
//
// This matches the Terraform provider's ucDirectoryPathSlashOnlySuppressDiff behavior.
// https://github.com/databricks/terraform-provider-databricks/blob/v1.65.1/catalog/resource_catalog.go#L57
func (*ResourceCatalog) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *catalog.CatalogInfo) error {
	if change.Action == deployplan.Skip {
		return nil
	}

	if path.String() != "storage_root" {
		return nil
	}

	newStr, newOk := change.New.(string)
	remoteStr, remoteOk := change.Remote.(string)
	if !newOk || !remoteOk {
		return nil
	}

	if newStr != remoteStr && strings.TrimRight(newStr, "/") == strings.TrimRight(remoteStr, "/") {
		change.Action = deployplan.Skip
		change.Reason = deployplan.ReasonURLNormalization
	}

	return nil
}
