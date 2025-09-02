package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceDatabaseCatalog struct {
	client *databricks.WorkspaceClient
	config database.DatabaseCatalog
}

func (d ResourceDatabaseCatalog) Config() any {
	return d.config
}

func (d *ResourceDatabaseCatalog) DoCreate(ctx context.Context) (string, error) {
	result, err := d.client.Database.CreateDatabaseCatalog(ctx, database.CreateDatabaseCatalogRequest{
		Catalog: d.config,
	})
	if err != nil {
		return "", err
	}
	return result.Name, nil
}

func (d ResourceDatabaseCatalog) DoUpdate(ctx context.Context, id string) error {
	request := database.UpdateDatabaseCatalogRequest{
		DatabaseCatalog: d.config,
		Name:            d.config.Name,
		UpdateMask:      "*",
	}

	_, err := d.client.Database.UpdateDatabaseCatalog(ctx, request)
	return err
}

func (d ResourceDatabaseCatalog) WaitAfterCreate(_ context.Context) error {
	return nil
}

func (d ResourceDatabaseCatalog) WaitAfterUpdate(_ context.Context) error {
	return nil
}

func NewResourceDatabaseCatalog(client *databricks.WorkspaceClient, resource *resources.DatabaseCatalog) (*ResourceDatabaseCatalog, error) {
	return &ResourceDatabaseCatalog{
		client: client,
		config: resource.DatabaseCatalog,
	}, nil
}

func DeleteDatabaseCatalog(ctx context.Context, client *databricks.WorkspaceClient, name string) error {
	return client.Database.DeleteDatabaseCatalog(ctx, database.DeleteDatabaseCatalogRequest{
		Name: name,
	})
}
