package dresources

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

// deleteSyncedTableTimeout caps the post-delete poll so a stuck teardown does not
// wait forever. Deletion tears down the backing sync pipeline, which normally
// completes within a minute. DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT shortens it further.
const deleteSyncedTableTimeout = 5 * time.Minute

// PostgresSyncedTableRemote is the return type for DoRead. It embeds
// SyncedTable_SyncedTableSpec so that all paths in StateType are valid paths in
// RemoteType, enabling drift detection for spec fields once the backend echoes
// spec on GET.
type PostgresSyncedTableRemote struct {
	postgres.SyncedTable_SyncedTableSpec

	SyncedTableId string `json:"synced_table_id,omitempty"`

	Name       string                                  `json:"name,omitempty"`
	Status     *postgres.SyncedTable_SyncedTableStatus `json:"status,omitempty"`
	Uid        string                                  `json:"uid,omitempty"`
	CreateTime *types.Time                             `json:"create_time,omitempty"`
}

type ResourcePostgresSyncedTable struct {
	client *sdkclients.Clients
}

type PostgresSyncedTableState = resources.PostgresSyncedTableConfig

func (*ResourcePostgresSyncedTable) New(client *sdkclients.Clients) *ResourcePostgresSyncedTable {
	return &ResourcePostgresSyncedTable{client: client}
}

func (*ResourcePostgresSyncedTable) PrepareState(input *resources.PostgresSyncedTable) *PostgresSyncedTableState {
	return &PostgresSyncedTableState{
		SyncedTableId:               input.SyncedTableId,
		SyncedTable_SyncedTableSpec: input.SyncedTable_SyncedTableSpec,
	}
}

func (*ResourcePostgresSyncedTable) RemapState(remote *PostgresSyncedTableRemote) *PostgresSyncedTableState {
	return &PostgresSyncedTableState{
		SyncedTableId:               remote.SyncedTableId,
		SyncedTable_SyncedTableSpec: remote.SyncedTable_SyncedTableSpec,
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
func makePostgresSyncedTableRemote(resp *sdkpostgres.SyncedTable) (*PostgresSyncedTableRemote, error) {
	syncedTable, err := postgres.SyncedTableFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.SyncedTable_SyncedTableSpec
	if syncedTable.Spec != nil {
		spec = *syncedTable.Spec
	}
	return &PostgresSyncedTableRemote{
		SyncedTable_SyncedTableSpec: spec,
		SyncedTableId:               strings.TrimPrefix(utils.Deref(syncedTable.Name), "synced_tables/"),
		Name:                        utils.Deref(syncedTable.Name),
		Status:                      syncedTable.Status,
		Uid:                         utils.Deref(syncedTable.Uid),
		CreateTime:                  syncedTable.CreateTime,
	}, nil
}

func (r *ResourcePostgresSyncedTable) DoRead(ctx context.Context, id string) (*PostgresSyncedTableRemote, error) {
	syncedTable, err := r.client.Postgres.GetSyncedTable(ctx, sdkpostgres.GetSyncedTableRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresSyncedTableRemote(syncedTable)
}

func (r *ResourcePostgresSyncedTable) DoCreate(ctx context.Context, config *PostgresSyncedTableState) (string, *PostgresSyncedTableRemote, error) {
	request, err := postgres.CreateSyncedTableRequestToSDK(&postgres.CreateSyncedTableRequest{
		SyncedTableId: nonEmpty(config.SyncedTableId),
		SyncedTable: &postgres.SyncedTable{
			Spec:          &config.SyncedTable_SyncedTableSpec,
			CreateTime:    nil, // Not set by DABs
			Name:          nil, // Not set by DABs
			Status:        nil, // Not set by DABs
			SyncedTableId: nil, // Not set by DABs
			Uid:           nil, // Not set by DABs
		},
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateSyncedTable(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	remote, err := makePostgresSyncedTableRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresSyncedTable) DoDelete(ctx context.Context, id string, _ *PostgresSyncedTableState) error {
	waiter, err := r.client.Postgres.DeleteSyncedTable(ctx, sdkpostgres.DeleteSyncedTableRequest{Name: new(id)})
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
		_, getErr := r.client.Postgres.GetSyncedTable(ctx, sdkpostgres.GetSyncedTableRequest{Name: new(id)})
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		return nil, retries.Continues("synced table still exists, waiting for deletion to complete")
	})
	return err
}
