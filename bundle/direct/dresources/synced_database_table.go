package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	database "github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
)

type ResourceSyncedDatabaseTable struct {
	client *sdkclients.Clients
}

func (*ResourceSyncedDatabaseTable) New(client *sdkclients.Clients) *ResourceSyncedDatabaseTable {
	return &ResourceSyncedDatabaseTable{client: client}
}

func (*ResourceSyncedDatabaseTable) PrepareState(input *resources.SyncedDatabaseTable) *database.SyncedDatabaseTable {
	return &input.SyncedDatabaseTable
}

func (r *ResourceSyncedDatabaseTable) DoRead(ctx context.Context, name string) (*database.SyncedDatabaseTable, error) {
	table, err := r.client.Database.GetSyncedDatabaseTable(ctx, sdkdatabase.GetSyncedDatabaseTableRequest{Name: new(name)})
	if err != nil {
		return nil, err
	}
	return database.SyncedDatabaseTableFromSDK(table)
}

func (r *ResourceSyncedDatabaseTable) DoCreate(ctx context.Context, config *database.SyncedDatabaseTable) (string, *database.SyncedDatabaseTable, error) {
	request, err := database.CreateSyncedDatabaseTableRequestToSDK(&database.CreateSyncedDatabaseTableRequest{SyncedTable: config})
	if err != nil {
		return "", nil, err
	}
	result, err := r.client.Database.CreateSyncedDatabaseTable(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(result.Name), nil, nil
}

func (r *ResourceSyncedDatabaseTable) DoDelete(ctx context.Context, id string, _ *database.SyncedDatabaseTable) error {
	return r.client.Database.DeleteSyncedDatabaseTable(ctx, sdkdatabase.DeleteSyncedDatabaseTableRequest{
		Name:      new(id),
		PurgeData: nil, // Not set by DABs
	})
}
