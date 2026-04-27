package dresources

import (
	"context"

	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/ucm/config/resources"
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

func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *catalog.CreateCatalog, _ *PlanEntry) (*catalog.CatalogInfo, error) {
	updateRequest := catalog.UpdateCatalog{
		Comment:                      config.Comment,
		EnablePredictiveOptimization: "",
		IsolationMode:                "",
		ManagedEncryptionSettings:    config.ManagedEncryptionSettings,
		Name:                         id,
		Options:                      config.Options,
		Owner:                        "",
		Properties:                   config.Properties,
		ForceSendFields:              utils.FilterFields[catalog.UpdateCatalog](config.ForceSendFields, "EnablePredictiveOptimization", "IsolationMode", "Owner"),
	}

	return r.client.Catalogs.Update(ctx, updateRequest)
}

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string) error {
	return r.client.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{
		Name:  id,
		Force: true,
	})
}
