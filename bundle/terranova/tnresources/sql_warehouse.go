package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
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

// PrepareConfig converts bundle config to the SDK type.
func (*ResourceSqlWarehouse) PrepareConfig(input *resources.SqlWarehouse) *sql.CreateWarehouseRequest {
	return &input.CreateWarehouseRequest
}

// DoCreate creates the warehouse and returns its id.
func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context, config *sql.CreateWarehouseRequest) (string, error) {
	waiter, err := r.client.Warehouses.Create(ctx, *config)
	if err != nil {
		return "", err
	}
	return waiter.Id, nil
}

// DoUpdate updates the warehouse in place.
func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string, config *sql.CreateWarehouseRequest) error {
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
		ForceSendFields:         filterFields[sql.EditWarehouseRequest](config.ForceSendFields),
	}

	waiter, err := r.client.Warehouses.Edit(ctx, request)
	if err != nil {
		return err
	}

	if waiter.Id != id {
		log.Warnf(ctx, "sql_warehouses: response contains unexpected id=%#v (expected %#v)", waiter.Id, id)
	}

	return nil
}

func (r *ResourceSqlWarehouse) DoDelete(ctx context.Context, oldID string) error {
	return r.client.Warehouses.DeleteById(ctx, oldID)
}
