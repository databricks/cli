package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceSyncedDatabaseTable struct {
	client *databricks.WorkspaceClient
}

func (*ResourceSyncedDatabaseTable) New(client *databricks.WorkspaceClient) *ResourceSyncedDatabaseTable {
	return &ResourceSyncedDatabaseTable{client: client}
}

func (*ResourceSyncedDatabaseTable) PrepareConfig(input *resources.SyncedDatabaseTable) *database.SyncedDatabaseTable {
	return &input.SyncedDatabaseTable
}

func (r *ResourceSyncedDatabaseTable) DoCreate(ctx context.Context, config *database.SyncedDatabaseTable) (string, error) {
	result, err := r.client.Database.CreateSyncedDatabaseTable(ctx, database.CreateSyncedDatabaseTableRequest{
		SyncedTable: *config,
	})
	if err != nil {
		return "", err
	}
	return result.Name, nil
}

func (r *ResourceSyncedDatabaseTable) DoUpdate(ctx context.Context, id string, config *database.SyncedDatabaseTable) error {
	request := database.UpdateSyncedDatabaseTableRequest{
		SyncedTable: *config,
		Name:        id,
		UpdateMask:  "*",
	}

	_, err := r.client.Database.UpdateSyncedDatabaseTable(ctx, request)
	return err
}

func (r *ResourceSyncedDatabaseTable) DoDelete(ctx context.Context, id string) error {
	return r.client.Database.DeleteSyncedDatabaseTable(ctx, database.DeleteSyncedDatabaseTableRequest{
		Name: id,
	})
}
