package dresources

import (
	"context"

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
	return &PostgresSyncedTableState{
		SyncedTableId: TrimSyncedTablesPrefix(remote.Name),

		// GET does not return the spec, only the status. Match the postgres_project /
		// postgres_branch pattern: return an empty (non-nil) spec so field-level diffing
		// works correctly and remote drift on spec fields is invisible.
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
