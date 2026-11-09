package dresources

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"sync"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

const catalogsPath = "/api/2.1/unity-catalog/catalogs"

// CatalogRemote is CatalogInfo plus the catalog's id, a UUID that UC returns but the API spec does
// not declare, so the SDK drops it. A rename keeps the id and a recreate changes it (verified on
// AWS, Azure and GCP).
type CatalogRemote struct {
	catalog.CatalogInfo

	Id string `json:"id,omitempty"`
}

// Custom marshaler needed because embedded CatalogInfo has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *CatalogRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s CatalogRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type CatalogState struct {
	catalog.CreateCatalog

	// Recorded by the framework from the remote state (internal_id_field).
	Id string `json:"id,omitempty"`
}

// Custom marshaler needed because embedded CreateCatalog has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *CatalogState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s CatalogState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceCatalog struct {
	client    *databricks.WorkspaceClient
	apiClient func() (*client.DatabricksClient, error)
}

func (*ResourceCatalog) New(w *databricks.WorkspaceClient) *ResourceCatalog {
	return &ResourceCatalog{
		client: w,
		apiClient: sync.OnceValues(func() (*client.DatabricksClient, error) {
			return client.New(w.Config)
		}),
	}
}

// do calls the catalogs API like the SDK does, but decodes the response into CatalogRemote:
// the SDK's CatalogInfo has no id.
func (r *ResourceCatalog) do(ctx context.Context, method, path string, request any) (*CatalogRemote, error) {
	apiClient, err := r.apiClient()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Accept": "application/json"}
	if request != nil {
		headers["Content-Type"] = "application/json"
	}
	maps.Copy(headers, auth.WorkspaceIDHeaders(r.client.Config))
	var response CatalogRemote
	err = apiClient.Do(ctx, method, path, headers, nil, request, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *CatalogState {
	return &CatalogState{
		CreateCatalog: input.CreateCatalog,
		Id:            "", // Recorded by the framework from the remote state (internal_id_field).
	}
}

func (*ResourceCatalog) RemapState(info *CatalogRemote) *CatalogState {
	state := &CatalogState{Id: info.Id}
	state.CreateCatalog = catalog.CreateCatalog{
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
	}
	return state
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*CatalogRemote, error) {
	return r.do(ctx, http.MethodGet, catalogsPath+"/"+id, nil)
}

func (r *ResourceCatalog) DoCreate(ctx context.Context, config *CatalogState) (string, *CatalogRemote, error) {
	response, err := r.do(ctx, http.MethodPost, catalogsPath, config.CreateCatalog)
	if err != nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

// IDFromState returns the name, which is the ID.
func (*ResourceCatalog) IDFromState(state *CatalogState) string {
	return state.Name
}

// See schemaForceSend. Verified against a real workspace: {"comment": ""} clears it.
//
// CustomMaxRetentionHours is force-sent here even though schemaForceSend deliberately omits it:
// that omission was for parity with the removed Terraform engine, which does not apply to catalogs
// anyway. Verified against a real workspace: {"custom_max_retention_hours": 0} clears it (an empty PATCH is otherwise
// rejected with "Nothing to update").
var catalogForceSend = []string{"Comment", "CustomMaxRetentionHours"}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (*CatalogRemote, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		CustomMaxRetentionHours:      config.CustomMaxRetentionHours,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                "", // Not supported by DABs
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Only set if name actually changes (see DoUpdateWithID)
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](append(slices.Clone(catalogForceSend), config.ForceSendFields...), "EnablePredictiveOptimization", "IsolationMode", "Owner"),
	}

	return r.do(ctx, http.MethodPatch, catalogsPath+"/"+id, updateRequest)
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *CatalogState, _ *PlanEntry) (string, *CatalogRemote, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		CustomMaxRetentionHours:      config.CustomMaxRetentionHours,
		EnablePredictiveOptimization: "", // Not supported by DABs
		IsolationMode:                "", // Not supported by DABs
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		NewName:                      "", // Initialized below if needed
		Options:                      config.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](append(slices.Clone(catalogForceSend), config.ForceSendFields...), "EnablePredictiveOptimization", "IsolationMode", "Owner"),
	}

	if config.Name != id {
		updateRequest.NewName = config.Name
	}

	response, err := r.do(ctx, http.MethodPatch, catalogsPath+"/"+id, updateRequest)
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
