package dresources

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// deleteSyncedTableTimeout caps the post-delete poll so a stuck teardown does not
// wait forever. Deletion tears down the backing sync pipeline, which normally
// completes within a minute.
const deleteSyncedTableTimeout = 5 * time.Minute

// PostgresSyncedTableRemote is the return type for DoRead. It embeds
// SyncedTableSyncedTableSpec so that all paths in StateType are valid paths in
// RemoteType, enabling drift detection for spec fields once the backend echoes
// spec on GET.
type PostgresSyncedTableRemote struct {
	postgres.SyncedTableSyncedTableSpec

	SyncedTableId string `json:"synced_table_id,omitempty"`

	Name       string                                 `json:"name,omitempty"`
	Status     *postgres.SyncedTableSyncedTableStatus `json:"status,omitempty"`
	Uid        string                                 `json:"uid,omitempty"`
	CreateTime *sdktime.Time                          `json:"create_time,omitempty"`
}

// Custom marshaler needed because embedded SyncedTableSyncedTableSpec has its own
// MarshalJSON which would otherwise take over and ignore the additional fields.
func (s *PostgresSyncedTableRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresSyncedTableRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

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

func (*ResourcePostgresSyncedTable) RemapState(remote *PostgresSyncedTableRemote) *PostgresSyncedTableState {
	return &PostgresSyncedTableState{
		SyncedTableId:              remote.SyncedTableId,
		SyncedTableSyncedTableSpec: remote.SyncedTableSyncedTableSpec,
	}
}

// makePostgresSyncedTableRemote converts the SDK SyncedTable into the embedded
// remote shape. GET does not echo spec today (only status is returned); the
// embedded spec fields stay at their zero values, and postgres_synced_tables.yml suppresses
// phantom drift via ignore_remote_changes with reason spec:input_only.
//
// The synced-table API doesn't expose the user-facing id as a named field. It
// only appears as the trailing component of remote.Name, so we strip the
// constant "synced_tables/" prefix.
func makePostgresSyncedTableRemote(syncedTable *postgres.SyncedTable) *PostgresSyncedTableRemote {
	var spec postgres.SyncedTableSyncedTableSpec
	if syncedTable.Spec != nil {
		spec = *syncedTable.Spec
	}
	return &PostgresSyncedTableRemote{
		SyncedTableSyncedTableSpec: spec,
		SyncedTableId:              strings.TrimPrefix(syncedTable.Name, "synced_tables/"),
		Name:                       syncedTable.Name,
		Status:                     syncedTable.Status,
		Uid:                        syncedTable.Uid,
		CreateTime:                 syncedTable.CreateTime,
	}
}

func (r *ResourcePostgresSyncedTable) DoRead(ctx context.Context, id string) (*PostgresSyncedTableRemote, error) {
	syncedTable, err := r.client.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresSyncedTableRemote(syncedTable), nil
}

func (r *ResourcePostgresSyncedTable) DoCreate(ctx context.Context, config *PostgresSyncedTableState) (string, *PostgresSyncedTableRemote, error) {
	waiter, err := r.client.Postgres.CreateSyncedTable(ctx, postgres.CreateSyncedTableRequest{
		SyncedTableId: config.SyncedTableId,
		SyncedTable: postgres.SyncedTable{
			Spec: &config.SyncedTableSyncedTableSpec,

			// Output-only fields.
			SyncedTableId:   "",
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
	remote := makePostgresSyncedTableRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresSyncedTable) DoDelete(ctx context.Context, id string, _ *PostgresSyncedTableState) error {
	waiter, err := r.client.Postgres.DeleteSyncedTable(ctx, postgres.DeleteSyncedTableRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}

// WaitAfterDelete polls GetSyncedTable until the table is gone. DeleteSyncedTable
// returns a synthetic operation immediately while the backend tears the table
// down asynchronously: GET keeps returning 200 with a DELETING provisioning
// state until the Unity Catalog record is finally removed, and only then returns
// 404. CreateSyncedTable checks existence against that same UC record, so a
// recreate's follow-up create issued during the window is rejected with 409
// ALREADY_EXISTS (observed on AWS). Because both the create's conflict check and
// this GET read the one UC record, waiting for GET to stop returning the table
// is enough to make the recreate safe.
//
// The poll surfaces the terminal read error: the framework maps a NotFound to
// success (the table is finally gone — see Adapter.WaitAfterDelete) and propagates
// anything else. So a poll timeout, an unexpected backend error, or a cancelled
// deploy fails the recreate rather than racing the create into a 409.
//
// The poll runs up to deleteSyncedTableTimeout.
func (r *ResourcePostgresSyncedTable) WaitAfterDelete(ctx context.Context, id string) error {
	_, err := retries.Poll[struct{}](ctx, deleteSyncedTableTimeout, func() (*struct{}, *retries.Err) {
		_, getErr := r.client.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: id})
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		return nil, retries.Continues("synced table still exists, waiting for deletion to complete")
	})
	return err
}
