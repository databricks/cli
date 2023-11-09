package cfgpickers

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/fatih/color"
)

var ErrNoCompatibleWarehouses = errors.New("no compatible warehouses")

type warehouseFilter func(sql.EndpointInfo) bool

func WithWarehouseTypes(types ...sql.EndpointInfoWarehouseType) func(sql.EndpointInfo) bool {
	allowed := map[sql.EndpointInfoWarehouseType]bool{}
	for _, v := range types {
		allowed[v] = true
	}
	return func(ei sql.EndpointInfo) bool {
		return allowed[ei.WarehouseType]
	}
}

func AskForWarehouse(ctx context.Context, w *databricks.WorkspaceClient, filters ...warehouseFilter) (string, error) {
	all, err := w.Warehouses.ListAll(ctx, sql.ListWarehousesRequest{})
	if err != nil {
		return "", fmt.Errorf("list warehouses: %w", err)
	}
	var lastWarehouseID string
	names := map[string]string{}
	for _, warehouse := range all {
		var skip bool
		for _, filter := range filters {
			if !filter(warehouse) {
				skip = true
			}
		}
		if skip {
			continue
		}
		var state string
		switch warehouse.State {
		case sql.StateRunning:
			state = color.GreenString(warehouse.State.String())
		case sql.StateStopped, sql.StateDeleted, sql.StateStopping, sql.StateDeleting:
			state = color.RedString(warehouse.State.String())
		default:
			state = color.BlueString(warehouse.State.String())
		}
		visibleTouser := fmt.Sprintf("%s (%s %s)", warehouse.Name, state, warehouse.WarehouseType)
		names[visibleTouser] = warehouse.Id
		lastWarehouseID = warehouse.Id
	}
	if len(names) == 0 {
		return "", ErrNoCompatibleWarehouses
	}
	if len(names) == 1 {
		return lastWarehouseID, nil
	}
	return cmdio.Select(ctx, names, "Choose SQL Warehouse")
}
