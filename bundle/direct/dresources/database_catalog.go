package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	database "github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
)

type ResourceDatabaseCatalog struct {
	client *sdkclients.Clients
}

func (*ResourceDatabaseCatalog) New(client *sdkclients.Clients) *ResourceDatabaseCatalog {
	return &ResourceDatabaseCatalog{client: client}
}

func (*ResourceDatabaseCatalog) PrepareState(input *resources.DatabaseCatalog) *database.DatabaseCatalog {
	return &input.DatabaseCatalog
}

func (r *ResourceDatabaseCatalog) DoRead(ctx context.Context, id string) (*database.DatabaseCatalog, error) {
	catalog, err := r.client.Database.GetDatabaseCatalog(ctx, sdkdatabase.GetDatabaseCatalogRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return database.DatabaseCatalogFromSDK(catalog)
}

func (r *ResourceDatabaseCatalog) DoCreate(ctx context.Context, config *database.DatabaseCatalog) (string, *database.DatabaseCatalog, error) {
	request, err := database.CreateDatabaseCatalogRequestToSDK(&database.CreateDatabaseCatalogRequest{Catalog: config})
	if err != nil {
		return "", nil, err
	}
	result, err := r.client.Database.CreateDatabaseCatalog(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(result.Name), nil, nil
}

func (r *ResourceDatabaseCatalog) DoDelete(ctx context.Context, id string, _ *database.DatabaseCatalog) error {
	return r.client.Database.DeleteDatabaseCatalog(ctx, sdkdatabase.DeleteDatabaseCatalogRequest{
		Name: new(id),
	})
}
