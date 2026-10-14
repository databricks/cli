package dresources

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	warehouses "github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"github.com/databricks/cli/libs/utils"
	sdkwarehouses "github.com/databricks/sdk-go/warehouses/v1"
)

// SqlWarehouseState is the state type for SqlWarehouse resources. It extends sql.CreateWarehouseRequest
// with lifecycle settings.
type SqlWarehouseState struct {
	warehouses.CreateWarehouseRequest

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`
}

type SqlWarehouseRemote struct {
	warehouses.GetWarehouseResponse

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`
}

type ResourceSqlWarehouse struct {
	client *sdkclients.Clients
}

func (*ResourceSqlWarehouse) New(client *sdkclients.Clients) *ResourceSqlWarehouse {
	return &ResourceSqlWarehouse{client: client}
}

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

func (*ResourceSqlWarehouse) RemapState(warehouse *SqlWarehouseRemote) *SqlWarehouseState {
	return &SqlWarehouseState{
		CreateWarehouseRequest: warehouses.CreateWarehouseRequest{
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
			StatementTimeout:        warehouse.StatementTimeout,
			Tags:                    warehouse.Tags,
			WarehouseType:           warehouse.WarehouseType,
		},
		Lifecycle: warehouse.Lifecycle,
	}
}

func (r *ResourceSqlWarehouse) DoRead(ctx context.Context, id string) (*SqlWarehouseRemote, error) {
	resp, err := r.client.Warehouses.GetWarehouse(ctx, sdkwarehouses.GetWarehouseRequest{Id: new(id)})
	if err != nil {
		return nil, err
	}
	warehouse, err := warehouses.GetWarehouseResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	remote := &SqlWarehouseRemote{
		GetWarehouseResponse: *warehouse,
		Lifecycle:            nil,
	}

	switch warehouse.State {
	case warehouses.EndpointState_Running:
		remote.Lifecycle = &StateLifecycle{Started: new(true)}
	case warehouses.EndpointState_Stopped:
		remote.Lifecycle = &StateLifecycle{Started: new(false)}
	default:
		remote.Lifecycle = nil
	}
	return remote, nil
}

func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context, config *SqlWarehouseState) (string, *SqlWarehouseRemote, error) {
	request, err := warehouses.CreateWarehouseRequestToSDK(&config.CreateWarehouseRequest)
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Warehouses.CreateWarehouse(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return waiter.GetId(), nil, nil
}

func hasWarehouseChanges(entry *PlanEntry) bool {
	return entry.Changes.HasChangeExcept("lifecycle", "lifecycle.started")
}

func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string, config *SqlWarehouseState, entry *PlanEntry) (*SqlWarehouseRemote, error) {
	edited := hasWarehouseChanges(entry)
	if edited {
		c := config.CreateWarehouseRequest
		request, err := warehouses.EditWarehouseRequestToSDK(&warehouses.EditWarehouseRequest{
			AutoStopMins:            c.AutoStopMins,
			Channel:                 c.Channel,
			ClusterSize:             c.ClusterSize,
			CreatorName:             c.CreatorName,
			EnablePhoton:            c.EnablePhoton,
			EnableServerlessCompute: c.EnableServerlessCompute,
			Id:                      new(id),
			InstanceProfileArn:      c.InstanceProfileArn,
			MaxNumClusters:          c.MaxNumClusters,
			MinNumClusters:          c.MinNumClusters,
			Name:                    c.Name,
			SpotInstancePolicy:      c.SpotInstancePolicy,
			StatementTimeout:        c.StatementTimeout,
			Tags:                    c.Tags,
			WarehouseType:           c.WarehouseType,
		})
		if err != nil {
			return nil, err
		}

		waiter, err := r.client.Warehouses.EditWarehouse(ctx, *request)
		if err != nil {
			return nil, err
		}

		if waiter.GetId() != id {
			log.Warnf(ctx, "sql_warehouses: response contains unexpected id=%#v (expected %#v)", waiter.GetId(), id)
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	alreadyRunning := remoteWarehouseIsRunning(entry)
	if edited {
		alreadyRunning = true
	}
	if desiredStarted && !alreadyRunning {
		_, err := r.client.Warehouses.StartWarehouse(ctx, sdkwarehouses.StartRequest{Id: new(id)})
		return nil, err
	} else if !desiredStarted && alreadyRunning {
		_, err := r.client.Warehouses.StopWarehouse(ctx, sdkwarehouses.StopRequest{Id: new(id)})
		return nil, err
	}

	return nil, nil
}

// waitWarehouseState polls until the warehouse reaches target. Reaching one of
// the failure states first is an error.
func (r *ResourceSqlWarehouse) waitWarehouseState(ctx context.Context, id string, target sdkwarehouses.EndpointState, failure ...sdkwarehouses.EndpointState) error {
	_, err := retries.Poll(ctx, 20*time.Minute, func() (*sdkwarehouses.GetWarehouseResponse, *retries.Err) {
		warehouse, err := r.client.Warehouses.GetWarehouse(ctx, sdkwarehouses.GetWarehouseRequest{Id: new(id)})
		if err != nil {
			return nil, retries.Halt(err)
		}
		status := warehouse.State
		statusMessage := fmt.Sprintf("current status: %s", status)
		if warehouse.Health != nil {
			statusMessage = utils.Deref(warehouse.Health.Summary)
		}
		if status == target {
			return warehouse, nil
		}
		if slices.Contains(failure, status) {
			return nil, retries.Halt(fmt.Errorf("failed to reach %s, got %s: %s", target, status, statusMessage))
		}
		return nil, retries.Continues(statusMessage)
	})
	return err
}

func (r *ResourceSqlWarehouse) waitRunning(ctx context.Context, id string) error {
	return r.waitWarehouseState(ctx, id, sdkwarehouses.EndpointState_Running, sdkwarehouses.EndpointState_Stopped, sdkwarehouses.EndpointState_Deleted)
}

func (r *ResourceSqlWarehouse) waitStopped(ctx context.Context, id string) error {
	return r.waitWarehouseState(ctx, id, sdkwarehouses.EndpointState_Stopped)
}

func (r *ResourceSqlWarehouse) WaitAfterUpdate(ctx context.Context, id string, config *SqlWarehouseState) (*SqlWarehouseRemote, error) {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	if *config.Lifecycle.Started {
		return nil, r.waitRunning(ctx, id)
	}
	return nil, r.waitStopped(ctx, id)
}

func (r *ResourceSqlWarehouse) WaitAfterCreate(ctx context.Context, id string, config *SqlWarehouseState) (*SqlWarehouseRemote, error) {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	if err := r.waitRunning(ctx, id); err != nil {
		return nil, err
	}

	if !*config.Lifecycle.Started {
		stopWaiter, err := r.client.Warehouses.StopWarehouse(ctx, sdkwarehouses.StopRequest{Id: new(id)})
		if err != nil {
			return nil, err
		}
		_, err = stopWaiter.Wait(ctx)
		return nil, err
	}

	return nil, nil
}

func (r *ResourceSqlWarehouse) DoDelete(ctx context.Context, oldID string, _ *SqlWarehouseState) error {
	_, err := r.client.Warehouses.DeleteWarehouse(ctx, sdkwarehouses.DeleteWarehouseRequest{Id: new(oldID)})
	return err
}

func remoteWarehouseIsRunning(entry *PlanEntry) bool {
	if entry.RemoteState == nil {
		return false
	}
	remote, ok := entry.RemoteState.(*SqlWarehouseRemote)
	if !ok {
		return false
	}
	return remote.State == warehouses.EndpointState_Running
}
