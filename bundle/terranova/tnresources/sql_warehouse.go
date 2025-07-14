package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
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

func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, oldID string) (string, error) {
	request := sql.EditWarehouseRequest{}
	err := copyViaJSON(&request, r.config)
	if err != nil {
		return "", err
	}
	request.Id = oldID

	waiter, err := r.client.Warehouses.Edit(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Warehouses.Edit", Err: err}
	}
	return waiter.Id, nil
}

func (r *ResourceSqlWarehouse) WaitAfterCreate(ctx context.Context) error {
	// No need to wait for sql warehouse to be ready after creation similar to clusters
	return nil
}

func (r *ResourceSqlWarehouse) WaitAfterUpdate(ctx context.Context) error {
	// No need to wait for sql warehouse to be ready after update similar to clusters
	return nil
}

func (r *ResourceSqlWarehouse) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

func DeleteSqlWarehouse(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error {
	err := client.Warehouses.DeleteById(ctx, oldID)
	if err != nil {
		return SDKError{Method: "Warehouses.DeleteById", Err: err}
	}
	return nil
}
