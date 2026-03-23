package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/fieldcopy"
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

// sqlWarehouseRemapCopy maps GetWarehouseResponse (remote GET response) to CreateWarehouseRequest (local state).
var sqlWarehouseRemapCopy fieldcopy.Copy[sql.GetWarehouseResponse, sql.CreateWarehouseRequest]

func (*ResourceSqlWarehouse) RemapState(warehouse *sql.GetWarehouseResponse) *sql.CreateWarehouseRequest {
	result := sqlWarehouseRemapCopy.Do(warehouse)
	// WarehouseType requires explicit conversion between different named string types.
	result.WarehouseType = sql.CreateWarehouseRequestWarehouseType(warehouse.WarehouseType)
	return &result
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

// sqlWarehouseEditCopy maps CreateWarehouseRequest (local state) to EditWarehouseRequest (API request).
var sqlWarehouseEditCopy fieldcopy.Copy[sql.CreateWarehouseRequest, sql.EditWarehouseRequest]

// DoUpdate updates the warehouse in place.
func (r *ResourceSqlWarehouse) DoUpdate(ctx context.Context, id string, config *sql.CreateWarehouseRequest, _ Changes) (*sql.GetWarehouseResponse, error) {
	request := sqlWarehouseEditCopy.Do(config)
	request.Id = id
	// WarehouseType requires explicit conversion between different named string types.
	request.WarehouseType = sql.EditWarehouseRequestWarehouseType(config.WarehouseType)

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

func init() {
	registerCopy(&sqlWarehouseRemapCopy)
	registerCopy(&sqlWarehouseEditCopy)
}
