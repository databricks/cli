package dresources

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

func (*ResourceSyncedDatabaseTable) PrepareState(input *resources.SyncedDatabaseTable) *database.SyncedDatabaseTable {
	return &input.SyncedDatabaseTable
}

func (r *ResourceSyncedDatabaseTable) DoRead(ctx context.Context, name string) (*database.SyncedDatabaseTable, error) {
	return r.client.Database.GetSyncedDatabaseTableByName(ctx, name)
}

func (r *ResourceSyncedDatabaseTable) DoCreate(ctx context.Context, config *database.SyncedDatabaseTable) (string, *database.SyncedDatabaseTable, error) {
	result, err := r.client.Database.CreateSyncedDatabaseTable(ctx, database.CreateSyncedDatabaseTableRequest{
		SyncedTable: *config,
	})
	if err != nil {
		return "", nil, err
	}
	return result.Name, nil, nil
}

func (r *ResourceSyncedDatabaseTable) DoDelete(ctx context.Context, id string, _ *database.SyncedDatabaseTable) error {
	return r.client.Database.DeleteSyncedDatabaseTable(ctx, database.DeleteSyncedDatabaseTableRequest{
		Name:            id,
		PurgeData:       false,
		ForceSendFields: nil,
	})
}
