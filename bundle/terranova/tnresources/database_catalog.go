package tnresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
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

func (d ResourceDatabaseCatalog) DoCreate(ctx context.Context) (string, error) {
	fmt.Printf("Creating catalog with name=%s", d.config.Name)
	response, err := d.client.Database.CreateDatabaseCatalog(ctx, database.CreateDatabaseCatalogRequest{
		Catalog: d.config,
	})
	if err != nil {
		return "", SDKError{Method: "Database.CreateDatabaseCatalog", Err: err}
	}
	return response.Name, nil
}

func (d ResourceDatabaseCatalog) DoUpdate(ctx context.Context, oldID string) (string, error) {
	panic("updating a database catalog is not yet supported")
}

func (d ResourceDatabaseCatalog) WaitAfterCreate(ctx context.Context) error {
	return nil
}

func (d ResourceDatabaseCatalog) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func (d ResourceDatabaseCatalog) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

func NewResourceDatabaseCatalog(client *databricks.WorkspaceClient, resource *resources.DatabaseCatalog) (*ResourceDatabaseCatalog, error) {
	return &ResourceDatabaseCatalog{
		client: client,
		config: resource.DatabaseCatalog,
	}, nil
}

func DeleteDatabaseCatalog(ctx context.Context, client *databricks.WorkspaceClient, oldName string) error {
	err := client.Database.DeleteDatabaseCatalog(ctx, database.DeleteDatabaseCatalogRequest{
		Name: oldName,
	})
	if err != nil {
		return SDKError{Method: "Database.DeleteDatabaseCatalog", Err: err}
	}
	return nil
}
