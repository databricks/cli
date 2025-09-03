package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceSyncedDatabaseTable struct {
	client *databricks.WorkspaceClient
	config database.SyncedDatabaseTable
}

func (s ResourceSyncedDatabaseTable) Config() any {
	return s.config
}

func (s *ResourceSyncedDatabaseTable) DoCreate(ctx context.Context) (string, error) {
	result, err := s.client.Database.CreateSyncedDatabaseTable(ctx, database.CreateSyncedDatabaseTableRequest{
		SyncedTable: s.config,
	})
	if err != nil {
		return "", err
	}
	return result.Name, nil
}

func (s ResourceSyncedDatabaseTable) DoUpdate(ctx context.Context, id string) error {
	request := database.UpdateSyncedDatabaseTableRequest{
		SyncedTable: s.config,
		Name:        s.config.Name,
		UpdateMask:  "*",
	}

	_, err := s.client.Database.UpdateSyncedDatabaseTable(ctx, request)
	return err
}

func (s ResourceSyncedDatabaseTable) WaitAfterCreate(_ context.Context) error {
	return nil
}

func (s ResourceSyncedDatabaseTable) WaitAfterUpdate(_ context.Context) error {
	return nil
}

func NewResourceSyncedDatabaseTable(client *databricks.WorkspaceClient, resource *resources.SyncedDatabaseTable) (*ResourceSyncedDatabaseTable, error) {
	return &ResourceSyncedDatabaseTable{
		client: client,
		config: resource.SyncedDatabaseTable,
	}, nil
}

func DeleteSyncedDatabaseTable(ctx context.Context, client *databricks.WorkspaceClient, name string) error {
	return client.Database.DeleteSyncedDatabaseTable(ctx, database.DeleteSyncedDatabaseTableRequest{
		Name: name,
	})
}
