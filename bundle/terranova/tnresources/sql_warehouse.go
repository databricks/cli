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
	config sql.CreateWarehouseRequest
}

func NewResourceSqlWarehouse(client *databricks.WorkspaceClient, resource *resources.SqlWarehouse) (*ResourceSqlWarehouse, error) {
	return &ResourceSqlWarehouse{
		client: client,
		config: resource.CreateWarehouseRequest,
	}, nil
}

func (r *ResourceSqlWarehouse) Config() any {
	return r.config
}

func (r *ResourceSqlWarehouse) DoCreate(ctx context.Context) (string, error) {
	waiter, err := r.client.Warehouses.Create(ctx, r.config)
	if err != nil {
		return "", SDKError{Method: "Warehouses.Create", Err: err}
	}

	return waiter.Id, nil
}

func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string) error {
	request := sql.EditWarehouseRequest{
		AutoStopMins:            r.config.AutoStopMins,
		Channel:                 r.config.Channel,
		ClusterSize:             r.config.ClusterSize,
		CreatorName:             r.config.CreatorName,
		EnablePhoton:            r.config.EnablePhoton,
		EnableServerlessCompute: r.config.EnableServerlessCompute,
		Id:                      id,
		InstanceProfileArn:      r.config.InstanceProfileArn,
		MaxNumClusters:          r.config.MaxNumClusters,
		MinNumClusters:          r.config.MinNumClusters,
		Name:                    r.config.Name,
		SpotInstancePolicy:      r.config.SpotInstancePolicy,
		Tags:                    r.config.Tags,
		WarehouseType:           sql.EditWarehouseRequestWarehouseType(r.config.WarehouseType),
		ForceSendFields:         filterFields[sql.EditWarehouseRequest](r.config.ForceSendFields),
	}

	waiter, err := r.client.Warehouses.Edit(ctx, request)
	if err != nil {
		return SDKError{Method: "Warehouses.Edit", Err: err}
	}

	if waiter.Id != id {
		log.Warnf(ctx, "sql_warehouses: response contains unexpected id=%#v (expected %#v)", waiter.Id, id)
	}

	return nil
}

func (r *ResourceSqlWarehouse) WaitAfterCreate(ctx context.Context) error {
	// No need to wait for sql warehouse to be ready after creation similar to clusters
	return nil
}

func (r *ResourceSqlWarehouse) WaitAfterUpdate(ctx context.Context) error {
	// No need to wait for sql warehouse to be ready after update similar to clusters
	return nil
}

func DeleteSqlWarehouse(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error {
	err := client.Warehouses.DeleteById(ctx, oldID)
	if err != nil {
		return SDKError{Method: "Warehouses.DeleteById", Err: err}
	}
	return nil
}
