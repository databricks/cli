package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// SqlWarehouseState is the state type for SqlWarehouse resources. It extends sql.CreateWarehouseRequest
// with lifecycle settings.
type SqlWarehouseState struct {
	sql.CreateWarehouseRequest

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`
}

// Custom marshalers needed because embedded sql.CreateWarehouseRequest has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *SqlWarehouseState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SqlWarehouseState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// SqlWarehouseRemote extends sql.GetWarehouseResponse with a synthetic Lifecycle field so that
// RemoteType satisfies TestRemoteSuperset (every field in SqlWarehouseState exists in SqlWarehouseRemote).
// Lifecycle.Started is populated by DoRead from the warehouse's running state.
type SqlWarehouseRemote struct {
	sql.GetWarehouseResponse

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`
}

func (r *SqlWarehouseRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r SqlWarehouseRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

type ResourceSqlWarehouse struct {
	client *databricks.WorkspaceClient
}

// New initializes a ResourceSqlWarehouse with the given client.
func (*ResourceSqlWarehouse) New(client *databricks.WorkspaceClient) *ResourceSqlWarehouse {
	return &ResourceSqlWarehouse{client: client}
}

// PrepareState converts bundle config to the SDK type.
func (*ResourceSqlWarehouse) PrepareState(input *resources.SqlWarehouse) *SqlWarehouseState {
	s := &SqlWarehouseState{
		CreateWarehouseRequest: input.CreateWarehouseRequest,
		Lifecycle:              nil,
	}
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &StateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote SqlWarehouseRemote to SqlWarehouseState for diff comparison.
// Started is derived from warehouse state so the planner can detect start/stop changes.
func (*ResourceSqlWarehouse) RemapState(warehouse *SqlWarehouseRemote) *SqlWarehouseState {
	started := warehouse.State == sql.StateRunning
	return &SqlWarehouseState{
		CreateWarehouseRequest: sql.CreateWarehouseRequest{
			AutoStopMins:            warehouse.AutoStopMins,
			Channel:                 warehouse.Channel,
			ClusterSize:             warehouse.ClusterSize,
			CreatorName:             warehouse.CreatorName,
			EnablePhoton:            warehouse.EnablePhoton,
			EnableServerlessCompute: warehouse.EnableServerlessCompute,
			InstanceProfileArn:      warehouse.InstanceProfileArn,
			MaxNumClusters:          warehouse.MaxNumClusters,
			MinNumClusters:          warehouse.MinNumClusters,
			Name:                    warehouse.Name,
			SpotInstancePolicy:      warehouse.SpotInstancePolicy,
			Tags:                    warehouse.Tags,
			WarehouseType:           sql.CreateWarehouseRequestWarehouseType(warehouse.WarehouseType),
			ForceSendFields:         utils.FilterFields[sql.CreateWarehouseRequest](warehouse.ForceSendFields),
		},
		Lifecycle: &StateLifecycle{Started: &started},
	}
}

// DoRead reads the warehouse by id.
func (r *ResourceSqlWarehouse) DoRead(ctx context.Context, id string) (*SqlWarehouseRemote, error) {
	warehouse, err := r.client.Warehouses.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	remote := &SqlWarehouseRemote{
		GetWarehouseResponse: *warehouse,
		Lifecycle:            nil,
	}

	switch warehouse.State {
	case sql.StateRunning:
		started := true
		remote.Lifecycle = &StateLifecycle{Started: &started}
	case sql.StateStopped:
		started := false
		remote.Lifecycle = &StateLifecycle{Started: &started}
	default:
		remote.Lifecycle = nil
	}
	return remote, nil
}

// DoCreate creates the warehouse and returns its id.
func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context, engine *StateSaver, config *SqlWarehouseState) (string, *SqlWarehouseRemote, error) {
	waiter, err := r.client.Warehouses.Create(ctx, config.CreateWarehouseRequest)
	if err != nil {
		return "", nil, err
	}
	id := waiter.Id

	// Save with Lifecycle=nil: warehouse exists but lifecycle has not been applied yet
	// (it always starts RUNNING). If the subsequent wait or stop is interrupted, the
	// planner sees a real diff (nil→desired) and re-applies lifecycle on the next deploy.
	SaveStateWith(ctx, engine, id, config, &config.Lifecycle, (*StateLifecycle)(nil))

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return id, nil, nil
	}

	// Always wait for RUNNING first: warehouses start asynchronously.
	_, err = r.client.Warehouses.WaitGetWarehouseRunning(ctx, id, 20*time.Minute, nil)
	if err != nil {
		return "", nil, err
	}

	if !*config.Lifecycle.Started {
		// started=false: stop the warehouse after it reaches RUNNING.
		stopWaiter, err := r.client.Warehouses.Stop(ctx, sql.StopRequest{Id: id})
		if err != nil {
			return "", nil, err
		}
		_, err = stopWaiter.Get()
		if err != nil {
			return "", nil, err
		}
	}

	return id, nil, nil
}

// hasWarehouseChanges reports whether the plan entry contains any Update changes
// to fields that belong to the Warehouse Edit API (i.e., not lifecycle-only fields).
func hasWarehouseChanges(entry *PlanEntry) bool {
	return entry.Changes.HasChangeExcept("lifecycle", "lifecycle.started")
}

// DoUpdate updates the warehouse in place.
func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, _ *StateSaver, id string, config *SqlWarehouseState, entry *PlanEntry) (*SqlWarehouseRemote, error) {
	if hasWarehouseChanges(entry) {
		request := sql.EditWarehouseRequest{
			AutoStopMins:            config.AutoStopMins,
			Channel:                 config.Channel,
			ClusterSize:             config.ClusterSize,
			CreatorName:             config.CreatorName,
			EnablePhoton:            config.EnablePhoton,
			EnableServerlessCompute: config.EnableServerlessCompute,
			Id:                      id,
			InstanceProfileArn:      config.InstanceProfileArn,
			MaxNumClusters:          config.MaxNumClusters,
			MinNumClusters:          config.MinNumClusters,
			Name:                    config.Name,
			SpotInstancePolicy:      config.SpotInstancePolicy,
			Tags:                    config.Tags,
			WarehouseType:           sql.EditWarehouseRequestWarehouseType(config.WarehouseType),
			ForceSendFields:         utils.FilterFields[sql.EditWarehouseRequest](config.ForceSendFields),
		}

		waiter, err := r.client.Warehouses.Edit(ctx, request)
		if err != nil {
			return nil, err
		}

		if waiter.Id != id {
			log.Warnf(ctx, "sql_warehouses: response contains unexpected id=%#v (expected %#v)", waiter.Id, id)
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	alreadyRunning := remoteWarehouseIsRunning(entry)
	if desiredStarted && !alreadyRunning {
		_, err := r.client.Warehouses.Start(ctx, sql.StartRequest{Id: id})
		if err != nil {
			return nil, err
		}
	} else if !desiredStarted && alreadyRunning {
		_, err := r.client.Warehouses.Stop(ctx, sql.StopRequest{Id: id})
		if err != nil {
			return nil, err
		}
	}

	if desiredStarted {
		_, err := r.client.Warehouses.WaitGetWarehouseRunning(ctx, id, 20*time.Minute, nil)
		return nil, err
	}
	_, err := r.client.Warehouses.WaitGetWarehouseStopped(ctx, id, 20*time.Minute, nil)
	return nil, err
}

func (r *ResourceSqlWarehouse) DoDelete(ctx context.Context, oldID string, _ *SqlWarehouseState) error {
	return r.client.Warehouses.DeleteById(ctx, oldID)
}

// remoteWarehouseIsRunning reads the warehouse running state from the plan entry's remote state.
func remoteWarehouseIsRunning(entry *PlanEntry) bool {
	if entry.RemoteState == nil {
		return false
	}
	remote, ok := entry.RemoteState.(*SqlWarehouseRemote)
	if !ok {
		return false
	}
	return remote.State == sql.StateRunning
}
