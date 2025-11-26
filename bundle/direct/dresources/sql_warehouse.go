package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/deployplan"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type ResourceSqlWarehouse struct {
	client *databricks.WorkspaceClient
}

// New initializes a ResourceSqlWarehouse with the given client.
func (*ResourceSqlWarehouse) New(client *databricks.WorkspaceClient) *ResourceSqlWarehouse {
	return &ResourceSqlWarehouse{client: client}
}

// PrepareState converts bundle config to the SDK type.
func (*ResourceSqlWarehouse) PrepareState(input *resources.SqlWarehouse) *sql.CreateWarehouseRequest {
	return &input.CreateWarehouseRequest
}

func (*ResourceSqlWarehouse) RemapState(warehouse *sql.GetWarehouseResponse) *sql.CreateWarehouseRequest {
	return &sql.CreateWarehouseRequest{
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
	}
}

// DoRead reads the warehouse by id.
func (r *ResourceSqlWarehouse) DoRead(ctx context.Context, id string) (*sql.GetWarehouseResponse, error) {
	return r.client.Warehouses.GetById(ctx, id)
}

// DoCreate creates the warehouse and returns its id.
func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context, config *sql.CreateWarehouseRequest) (string, *sql.GetWarehouseResponse, error) {
	waiter, err := r.client.Warehouses.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return waiter.Id, nil, nil
}

// DoUpdate updates the warehouse in place.
func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string, config *sql.CreateWarehouseRequest, _ *deployplan.Changes) (*sql.GetWarehouseResponse, error) {
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

	return nil, nil
}

func (r *ResourceSqlWarehouse) DoDelete(ctx context.Context, oldID string) error {
	return r.client.Warehouses.DeleteById(ctx, oldID)
}
