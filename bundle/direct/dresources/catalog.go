package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// catalogState extends CreateCatalog with fields that can only be set via update.
type catalogState struct {
	catalog.CreateCatalog

	EnablePredictiveOptimization catalog.EnablePredictiveOptimization `json:"enable_predictive_optimization,omitempty"`
	IsolationMode                catalog.CatalogIsolationMode         `json:"isolation_mode,omitempty"`
	Owner                        string                               `json:"owner,omitempty"`
}

// Custom marshaling is required because CreateCatalog has custom marshaling that would otherwise swallow the extra fields.
func (s *catalogState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s catalogState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceCatalog struct {
	client *databricks.WorkspaceClient
}

func (*ResourceCatalog) New(client *databricks.WorkspaceClient) *ResourceCatalog {
	return &ResourceCatalog{client: client}
}

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *catalogState {
	return &catalogState{
		CreateCatalog:                input.CreateCatalog,
		EnablePredictiveOptimization: input.EnablePredictiveOptimization,
		IsolationMode:                input.IsolationMode,
		Owner:                        input.Owner,
	}
}

func (*ResourceCatalog) RemapState(info *catalog.CatalogInfo) *catalogState {
	return &catalogState{
		CreateCatalog: catalog.CreateCatalog{
			Comment:         info.Comment,
			ConnectionName:  info.ConnectionName,
			Name:            info.Name,
			Options:         info.Options,
			Properties:      info.Properties,
			ProviderName:    info.ProviderName,
			ShareName:       info.ShareName,
			StorageRoot:     info.StorageRoot,
			ForceSendFields: utils.FilterFields[catalog.CreateCatalog](info.ForceSendFields),
		},
		EnablePredictiveOptimization: info.EnablePredictiveOptimization,
		IsolationMode:                info.IsolationMode,
		Owner:                        info.Owner,
	}
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*catalog.CatalogInfo, error) {
	return r.client.Catalogs.GetByName(ctx, id)
}

// DoCreate creates the catalog and applies update-only fields if set.
func (r *ResourceCatalog) DoCreate(ctx context.Context, config *catalogState) (string, *catalog.CatalogInfo, error) {
	response, err := r.client.Catalogs.Create(ctx, config.CreateCatalog)
	if err != nil || response == nil {
		return "", nil, err
	}

	// IsolationMode, EnablePredictiveOptimization, and Owner cannot be set during creation; apply them via update.
	if config.IsolationMode != "" || config.EnablePredictiveOptimization != "" || config.Owner != "" {
		response, err = r.applyUpdate(ctx, response.Name, "", config)
		if err != nil {
			return "", nil, err
		}
	}

	return response.Name, response, nil
}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *catalogState, _ Changes) (*catalog.CatalogInfo, error) {
	return r.applyUpdate(ctx, id, "", config)
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *catalogState) (string, *catalog.CatalogInfo, error) {
	newName := ""
	if config.Name != id {
		newName = config.Name
	}

	response, err := r.applyUpdate(ctx, id, newName, config)
	if err != nil {
		return "", nil, err
	}

	newID := id
	if newName != "" {
		newID = newName
	}

	return newID, response, nil
}

// applyUpdate builds and sends an UpdateCatalog request. newName is set only when renaming.
func (r *ResourceCatalog) applyUpdate(ctx context.Context, id, newName string, config *catalogState) (*catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: config.EnablePredictiveOptimization,
		IsolationMode:                config.IsolationMode,
		Name:                         id,
		NewName:                      newName,
		Options:                      config.Options,
		Owner:                        config.Owner,
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](config.ForceSendFields),
	}

	return r.client.Catalogs.Update(ctx, updateRequest)
}

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string) error {
	return r.client.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{
		Name:  id,
		Force: true,
	})
}
