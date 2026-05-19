package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresSyncedTable struct {
	client *databricks.WorkspaceClient
}

type PostgresSyncedTableState = resources.PostgresSyncedTableConfig

func (*ResourcePostgresSyncedTable) New(client *databricks.WorkspaceClient) *ResourcePostgresSyncedTable {
	return &ResourcePostgresSyncedTable{client: client}
}

func (*ResourcePostgresSyncedTable) PrepareState(input *resources.PostgresSyncedTable) *PostgresSyncedTableState {
	return &PostgresSyncedTableState{
		SyncedTableId:              input.SyncedTableId,
		SyncedTableSyncedTableSpec: input.SyncedTableSyncedTableSpec,
	}
}

func (*ResourcePostgresSyncedTable) RemapState(remote *postgres.SyncedTable) *PostgresSyncedTableState {
	// Unlike postgres_catalogs (which has Status.CatalogId), the synced-table
	// API doesn't expose the user-facing id as a named field. It only appears
	// as the trailing component of remote.Name, so we strip the constant
	// "synced_tables/" prefix.
	//
	// GET does not return the spec today (only status). Return an empty spec
	// and rely on the spec:input_only classifications generated from the
	// OpenAPI schema to suppress phantom drift until the backend starts
	// echoing spec values on GET.
	return &PostgresSyncedTableState{
		SyncedTableId: strings.TrimPrefix(remote.Name, "synced_tables/"),
		SyncedTableSyncedTableSpec: postgres.SyncedTableSyncedTableSpec{
			Branch:                         "",
			CreateDatabaseObjectsIfMissing: false,
			ExistingPipelineId:             "",
			NewPipelineSpec:                nil,
			PostgresDatabase:               "",
			PrimaryKeyColumns:              nil,
			SchedulingPolicy:               "",
			SourceTableFullName:            "",
			TimeseriesKey:                  "",
			ForceSendFields:                nil,
		},
	}
}

func (r *ResourcePostgresSyncedTable) DoRead(ctx context.Context, id string) (*postgres.SyncedTable, error) {
	return r.client.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: id})
}

func (r *ResourcePostgresSyncedTable) DoCreate(ctx context.Context, config *PostgresSyncedTableState) (string, *postgres.SyncedTable, error) {
	waiter, err := r.client.Postgres.CreateSyncedTable(ctx, postgres.CreateSyncedTableRequest{
		SyncedTableId: config.SyncedTableId,
		SyncedTable: postgres.SyncedTable{
			Spec: &config.SyncedTableSyncedTableSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Status:          nil,
			Uid:             "",
			ForceSendFields: nil,
		},
	})
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	return result.Name, result, nil
}

func (r *ResourcePostgresSyncedTable) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteSyncedTable(ctx, postgres.DeleteSyncedTableRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
