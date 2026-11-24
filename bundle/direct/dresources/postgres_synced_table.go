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
// completes within a minute. DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT shortens it further.
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
	req := postgres.CreateSyncedTableRequest{
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
	}
	start := time.Now()
	waiter, err := r.client.Postgres.CreateSyncedTable(ctx, req)
	if err != nil {
		return r.recoverFailedCreate(ctx, req, start, err)
	}
	remote, err := r.waitCreated(ctx, waiter)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// waitCreated waits for the create operation of a synced table that a create request returned.
func (r *ResourcePostgresSyncedTable) waitCreated(ctx context.Context, waiter postgres.CreateSyncedTableOperationInterface) (*PostgresSyncedTableRemote, error) {
	result, err := waitCreateOperation(ctx, waiter.Wait)
	if err != nil {
		return nil, err
	}
	return makePostgresSyncedTableRemote(result), nil
}

// recoverFailedCreate handles a create request that failed; see recoverCreate. A synced table that
// is still being deleted (see WaitAfterDelete) predates the request, so it is not taken over.
func (r *ResourcePostgresSyncedTable) recoverFailedCreate(ctx context.Context, req postgres.CreateSyncedTableRequest, start time.Time, createErr error) (string, *PostgresSyncedTableRemote, error) {
	if req.SyncedTableId == "" {
		return "", nil, createErr
	}

	name := "synced_tables/" + req.SyncedTableId
	remote, err := recoverCreate(ctx, createRecovery[postgres.SyncedTable, *PostgresSyncedTableRemote]{
		name:  "synced table " + name,
		start: start,
		get: func(ctx context.Context) (*postgres.SyncedTable, error) {
			return r.client.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: name})
		},
		createTime:      func(found *postgres.SyncedTable) (time.Time, bool) { return sdkTime(found.CreateTime) },
		alreadyExists:   isAlreadyExists,
		createdByCaller: nil,
		use: func(_ context.Context, found *postgres.SyncedTable) (*PostgresSyncedTableRemote, error) {
			return makePostgresSyncedTableRemote(found), nil
		},
		create: func(ctx context.Context) (func() (*PostgresSyncedTableRemote, error), error) {
			waiter, err := r.client.Postgres.CreateSyncedTable(ctx, req)
			if err != nil {
				return nil, err
			}
			return func() (*PostgresSyncedTableRemote, error) { return r.waitCreated(ctx, waiter) }, nil
		},
	}, createErr)
	if err != nil {
		return "", nil, err
	}
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
// The poll runs up to deleteSyncedTableTimeout, shortened to RESOURCE_MAX_WAIT when
// that is smaller. The engine resolves that env var once and passes it via context
// (WithResourceMaxWait); the general delete-wait in apply.go stays uncapped, so only
// this resource bounds itself by it.
func (r *ResourcePostgresSyncedTable) WaitAfterDelete(ctx context.Context, id string) error {
	timeout := deleteSyncedTableTimeout
	if maxWait, ok := resourceMaxWait(ctx); ok && maxWait > 0 {
		timeout = min(timeout, maxWait)
	}

	_, err := retries.Poll[struct{}](ctx, timeout, func() (*struct{}, *retries.Err) {
		_, getErr := r.client.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: id})
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		return nil, retries.Continues("synced table still exists, waiting for deletion to complete")
	})
	return err
}
