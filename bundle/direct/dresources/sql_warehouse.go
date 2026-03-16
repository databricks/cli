package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type SqlWarehouseState struct {
	sql.CreateWarehouseRequest
	Started *bool `json:"started,omitempty"`
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
	return &SqlWarehouseState{
		CreateWarehouseRequest: input.CreateWarehouseRequest,
		Started:                input.Lifecycle.Started,
	}
}

func (*ResourceSqlWarehouse) RemapState(warehouse *sql.GetWarehouseResponse) *SqlWarehouseState {
	return &SqlWarehouseState{Started: nil, CreateWarehouseRequest: sql.CreateWarehouseRequest{
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
	}}
}

// DoRead reads the warehouse by id.
func (r *ResourceSqlWarehouse) DoRead(ctx context.Context, id string) (*sql.GetWarehouseResponse, error) {
	return r.client.Warehouses.GetById(ctx, id)
}

// DoCreate creates the warehouse and returns its id.
func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context, config *SqlWarehouseState) (string, *sql.GetWarehouseResponse, error) {
	waiter, err := r.client.Warehouses.Create(ctx, config.CreateWarehouseRequest)
	if err != nil {
		return "", nil, err
	}
	switch {
	case config.Started != nil && *config.Started:
		// lifecycle.started=true: wait for the warehouse to reach the running state.
		warehouse, err := waiter.Get()
		if err != nil {
			return "", nil, err
		}
		return warehouse.Id, warehouse, nil
	case config.Started != nil && !*config.Started:
		// lifecycle.started=false: wait for running, then stop to reach stopped state.
		warehouse, err := waiter.Get()
		if err != nil {
			return "", nil, err
		}
		stopWait, err := r.client.Warehouses.Stop(ctx, sql.StopRequest{Id: warehouse.Id})
		if err != nil {
			return "", nil, fmt.Errorf("failed to stop warehouse %s: %w", warehouse.Id, err)
		}
		if _, err = stopWait.Get(); err != nil {
			return "", nil, fmt.Errorf("failed to wait for warehouse %s to stop: %w", warehouse.Id, err)
		}
		return warehouse.Id, nil, nil
	default:
		// lifecycle.started omitted: default behaviour, return immediately without waiting.
		return waiter.Id, nil, nil
	}
}

// DoUpdate updates the warehouse in place.
func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string, config *SqlWarehouseState, _ Changes) (*sql.GetWarehouseResponse, error) {
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

	if config.Started == nil {
		return nil, nil
	}

	warehouse, err := r.client.Warehouses.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse %s: %w", id, err)
	}

	if *config.Started {
		// lifecycle.started=true: ensure the warehouse is running.
		if warehouse.State == sql.StateStopped {
			startWait, err := r.client.Warehouses.Start(ctx, sql.StartRequest{Id: id})
			if err != nil {
				return nil, fmt.Errorf("failed to start warehouse %s: %w", id, err)
			}
			if _, err = startWait.Get(); err != nil {
				return nil, fmt.Errorf("failed to wait for warehouse %s to start: %w", id, err)
			}
		}
	} else {
		// lifecycle.started=false: ensure the warehouse is stopped.
		if warehouse.State != sql.StateStopped && warehouse.State != sql.StateStopping {
			stopWait, err := r.client.Warehouses.Stop(ctx, sql.StopRequest{Id: id})
			if err != nil {
				return nil, fmt.Errorf("failed to stop warehouse %s: %w", id, err)
			}
			if _, err = stopWait.Get(); err != nil {
				return nil, fmt.Errorf("failed to wait for warehouse %s to stop: %w", id, err)
			}
		}
	}

	return nil, nil
}

func (r *ResourceSqlWarehouse) DoDelete(ctx context.Context, oldID string) error {
	return r.client.Warehouses.DeleteById(ctx, oldID)
}

func (*ResourceSqlWarehouse) OverrideChangeDesc(_ context.Context, p *structpath.PathNode, change *ChangeDesc, _ *sql.GetWarehouseResponse) error {
	if change.Action == deployplan.Update && p.Prefix(1).String() == "started" {
		// started is lifecycle metadata, not an actual warehouse property.
		change.Action = deployplan.Skip
	}
	return nil
}
