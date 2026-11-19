package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structcopy"
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

var toCatalogState = newCopier[catalog.CatalogInfo, CatalogState]()

func (*ResourceCatalog) RemapState(info *catalog.CatalogInfo) *CatalogState {
	return toCatalogState.Copy(info)
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
	// It carries nothing else: fields are set one by one so that a field the SDK adds to
	// UpdateCatalog needs no change here.
	var setIsolationMode catalog.UpdateCatalog
	setIsolationMode.Name = response.Name
	setIsolationMode.IsolationMode = config.IsolationMode
	response, err = r.client.Catalogs.Update(ctx, setIsolationMode)
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

// Name addresses the catalog, so it is the id; the name in the config goes in NewName when it differs.
// EnablePredictiveOptimization and Owner are not supported by DABs.
var toUpdateCatalog = newCopier[CatalogState, catalog.UpdateCatalog](
	structcopy.Skip("Name"),
	structcopy.ForceSend(catalogForceSend...),
)

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (*catalog.CatalogInfo, error) {
	updateRequest := toUpdateCatalog.Copy(config)
	updateRequest.Name = id

	response, err := r.client.Catalogs.Update(ctx, *updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (string, *catalog.CatalogInfo, error) {
	updateRequest := toUpdateCatalog.Copy(config)
	updateRequest.Name = id

	if config.Name != id {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.Catalogs.Update(ctx, *updateRequest)
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
