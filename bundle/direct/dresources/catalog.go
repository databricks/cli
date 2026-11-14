package dresources

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// CatalogState is CreateCatalog plus isolation_mode, which the Create API does not
// accept; DoCreate applies it with a follow-up update.
type CatalogState struct {
	catalog.CreateCatalog
	IsolationMode catalog.CatalogIsolationMode `json:"isolation_mode,omitempty"`
}

// Custom marshalers needed because embedded catalog.CreateCatalog has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *CatalogState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s CatalogState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceCatalog struct {
	client *databricks.WorkspaceClient
}

func (*ResourceCatalog) New(client *databricks.WorkspaceClient) *ResourceCatalog {
	return &ResourceCatalog{client: client}
}

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *CatalogState {
	return &CatalogState{
		CreateCatalog: input.CreateCatalog,
		IsolationMode: input.IsolationMode,
	}
}

func (*ResourceCatalog) RemapState(info *catalog.CatalogInfo) *CatalogState {
	return &CatalogState{
		Comment:                   info.Comment,
		ConnectionName:            info.ConnectionName,
		CustomMaxRetentionHours:   info.CustomMaxRetentionHours,
		ManagedEncryptionSettings: info.ManagedEncryptionSettings,
		Name:                      info.Name,
		Options:                   info.Options,
		Properties:                info.Properties,
		ProviderName:              info.ProviderName,
		ShareName:                 info.ShareName,
		StorageRoot:               info.StorageRoot,
		ForceSendFields:           utils.FilterFields[catalog.CreateCatalog](info.ForceSendFields),
		IsolationMode:             info.IsolationMode,
	}
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*catalog.CatalogInfo, error) {
	return r.client.Catalogs.GetByName(ctx, id)
}

func (r *ResourceCatalog) DoCreate(ctx context.Context, config *CatalogState) (string, *catalog.CatalogInfo, error) {
	response, err := r.client.Catalogs.Create(ctx, config.CreateCatalog)
	if err != nil || response == nil {
		return "", nil, err
	}
	if config.IsolationMode == "" {
		return response.Name, response, nil
	}

	// UC only accepts isolation_mode on update, so a configured value takes a second call.
	response, err = r.client.Catalogs.Update(ctx, catalog.UpdateCatalog{
		Comment:                      "",
		CustomMaxRetentionHours:      0,
		EnablePredictiveOptimization: "",
		IsolationMode:                config.IsolationMode,
		ManagedEncryptionSettings:    nil,
		Name:                         response.Name,
		NewName:                      "",
		Options:                      nil,
		Owner:                        "",
		Properties:                   nil,
		ForceSendFields:              nil,
	})
	if err != nil {
		// The catalog exists but is not in state, so the next deploy fails with "already
		// exists"; name the cause here since the caller only knows a create failed.
		return "", nil, fmt.Errorf("catalog %s was created but setting isolation_mode failed: %w", config.Name, err)
	}
	return response.Name, response, nil
}

// See schemaForceSend. Verified against a real workspace: {"comment": ""} clears it.
//
// CustomMaxRetentionHours is force-sent here even though schemaForceSend deliberately omits it:
// that omission was for parity with the removed Terraform engine, which does not apply to catalogs
// anyway. Verified against a real workspace: {"custom_max_retention_hours": 0} clears it (an empty PATCH is otherwise
// rejected with "Nothing to update").
var catalogForceSend = []string{"Comment", "CustomMaxRetentionHours"}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (*catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		CustomMaxRetentionHours:      config.CustomMaxRetentionHours,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                config.IsolationMode,
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Only set if name actually changes (see DoUpdateWithID)
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](append(slices.Clone(catalogForceSend), config.ForceSendFields...), "EnablePredictiveOptimization", "Owner"),
	}

	response, err := r.client.Catalogs.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (string, *catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		CustomMaxRetentionHours:      config.CustomMaxRetentionHours,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                config.IsolationMode,
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Initialized below if needed
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](append(slices.Clone(catalogForceSend), config.ForceSendFields...), "EnablePredictiveOptimization", "Owner"),
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

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string, _ *CatalogState) error {
	return r.client.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{
		Name:            id,
		Force:           true,
		ForceSendFields: nil,
	})
}
