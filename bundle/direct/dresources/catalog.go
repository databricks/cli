package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	catalogs "github.com/databricks/cli/libs/sdktypes/uc/catalogs/v1"
	"github.com/databricks/cli/libs/utils"
	sdkcatalogs "github.com/databricks/sdk-go/uc/catalogs/v1"
)

type ResourceCatalog struct {
	client *sdkclients.Clients
}

func (*ResourceCatalog) New(client *sdkclients.Clients) *ResourceCatalog {
	return &ResourceCatalog{client: client}
}

func (*ResourceCatalog) PrepareState(input *resources.Catalog) *resources.CreateCatalog {
	return &input.CreateCatalog
}

func (*ResourceCatalog) RemapState(info *catalogs.CatalogInfo) *resources.CreateCatalog {
	return &resources.CreateCatalog{
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
	}
}

func (r *ResourceCatalog) DoRead(ctx context.Context, id string) (*catalogs.CatalogInfo, error) {
	info, err := r.client.Catalogs.GetCatalog(ctx, sdkcatalogs.GetCatalogRequest{NameArg: new(id)})
	if err != nil {
		return nil, err
	}
	return catalogs.CatalogInfoFromSDK(info)
}

func (r *ResourceCatalog) DoCreate(ctx context.Context, config *resources.CreateCatalog) (string, *catalogs.CatalogInfo, error) {
	request, err := toSDK[sdkcatalogs.CreateCatalogRequest](config.AsRequest())
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Catalogs.CreateCatalog(ctx, request)
	if err != nil || response == nil {
		return "", nil, err
	}
	info, err := catalogs.CatalogInfoFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(info.Name), info, nil
}

// DoUpdate updates the catalog in place and returns remote state.
func (r *ResourceCatalog) DoUpdate(ctx context.Context, id string, config *resources.CreateCatalog, _ *PlanEntry) (*catalogs.CatalogInfo, error) {
	return r.update(ctx, id, config, nil)
}

// DoUpdateWithID updates the catalog and returns the new ID if the name changes.
func (r *ResourceCatalog) DoUpdateWithID(ctx context.Context, id string, config *resources.CreateCatalog, _ *PlanEntry) (string, *catalogs.CatalogInfo, error) {
	var newName *string
	if name := utils.Deref(config.Name); name != id {
		newName = new(name)
	}
	info, err := r.update(ctx, id, config, newName)
	if err != nil {
		return "", nil, err
	}
	if newName != nil {
		return *newName, info, nil
	}
	return id, info, nil
}

func (r *ResourceCatalog) update(ctx context.Context, id string, config *resources.CreateCatalog, newName *string) (*catalogs.CatalogInfo, error) {
	// Owner, isolation_mode and enable_predictive_optimization are not supported by DABs.
	request, err := toSDK[sdkcatalogs.UpdateCatalogRequest](&catalogs.UpdateCatalogRequest{
		NameArg: new(id),
		NewName: newName,
		// Always sent so that removing them from config clears them. Verified against a real
		// workspace: {"comment": ""} and {"custom_max_retention_hours": 0} clear the values (an
		// empty PATCH is otherwise rejected with "Nothing to update").
		Comment:                   orZero(config.Comment),
		CustomMaxRetentionHours:   orZero(config.CustomMaxRetentionHours),
		ManagedEncryptionSettings: config.ManagedEncryptionSettings,
		Options:                   config.Options,
		Properties:                config.Properties,
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.Catalogs.UpdateCatalog(ctx, request)
	if err != nil {
		return nil, err
	}
	return catalogs.CatalogInfoFromSDK(response)
}

func (r *ResourceCatalog) DoDelete(ctx context.Context, id string, _ *resources.CreateCatalog) error {
	_, err := r.client.Catalogs.DeleteCatalog(ctx, sdkcatalogs.DeleteCatalogRequest{
		NameArg: new(id),
		Force:   new(true),
	})
	return err
}
