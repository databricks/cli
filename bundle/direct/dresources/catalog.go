package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
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

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *resources.Catalog {
	return input
}

func (*ResourceCatalog) RemapState(info *catalog.CatalogInfo) *resources.Catalog {
	return &resources.Catalog{
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
	}
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*catalog.CatalogInfo, error) {
	return r.client.Catalogs.GetByName(ctx, id)
}

func (r *ResourceCatalog) DoCreate(ctx context.Context, input *resources.Catalog) (string, *catalog.CatalogInfo, error) {
	response, err := r.client.Catalogs.Create(ctx, input.CreateCatalog)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, input *resources.Catalog, _ Changes) (*catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      input.Comment,
		EnablePredictiveOptimization: input.EnablePredictiveOptimization,
		IsolationMode:                input.IsolationMode,
		Name:                         id,
		NewName:                      "", // We recreate catalogs on name change intentionally.
		Options:                      input.Options,
		Owner:                        "", // Not supported by DABs
		Properties:                   input.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](input.ForceSendFields, "NewName", "Owner"),
	}

	response, err := r.client.Catalogs.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	if response != nil && response.Name != id {
		log.Warnf(ctx, "catalogs: response contains unexpected name=%#v (expected %#v)", response.Name, id)
	}

	return response, nil
}

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string) error {
	return r.client.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{
		Name:            id,
		Force:           true,
		ForceSendFields: nil,
	})
}
