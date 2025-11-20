package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceDatabaseCatalog struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDatabaseCatalog) New(client *databricks.WorkspaceClient) *ResourceDatabaseCatalog {
	return &ResourceDatabaseCatalog{client: client}
}

func (*ResourceDatabaseCatalog) PrepareState(input *resources.DatabaseCatalog) *database.DatabaseCatalog {
	return &input.DatabaseCatalog
}

func (r *ResourceDatabaseCatalog) DoRead(ctx context.Context, id string) (*database.DatabaseCatalog, error) {
	return r.client.Database.GetDatabaseCatalogByName(ctx, id)
}

func (r *ResourceDatabaseCatalog) DoCreate(ctx context.Context, config *database.DatabaseCatalog) (string, *database.DatabaseCatalog, error) {
	result, err := r.client.Database.CreateDatabaseCatalog(ctx, database.CreateDatabaseCatalogRequest{
		Catalog: *config,
	})
	if err != nil {
		return "", nil, err
	}
	return result.Name, nil, nil
}

func (r *ResourceDatabaseCatalog) DoUpdate(ctx context.Context, id string, config *database.DatabaseCatalog) (*database.DatabaseCatalog, error) {
	request := database.UpdateDatabaseCatalogRequest{
		DatabaseCatalog: *config,
		Name:            id,
		UpdateMask:      "*",
	}

	_, err := r.client.Database.UpdateDatabaseCatalog(ctx, request)
	return nil, err
}

func (r *ResourceDatabaseCatalog) DoDelete(ctx context.Context, id string) error {
	return r.client.Database.DeleteDatabaseCatalog(ctx, database.DeleteDatabaseCatalogRequest{
		Name: id,
	})
}
