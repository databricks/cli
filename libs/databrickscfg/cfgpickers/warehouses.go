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
	for _, v := range all {
		var skip bool
		for _, filter := range filters {
			if !filter(v) {
				skip = true
			}
		}
		if skip {
			continue
		}
		var state string
		switch v.State {
		case sql.StateRunning:
			state = color.GreenString(v.State.String())
		case sql.StateStopped, sql.StateDeleted, sql.StateStopping, sql.StateDeleting:
			state = color.RedString(v.State.String())
		default:
			state = color.BlueString(v.State.String())
		}
		visibleTouser := fmt.Sprintf("%s (%s %s)", v.Name, state, v.WarehouseType)
		names[visibleTouser] = v.Id
		lastWarehouseID = v.Id
	}
	if len(names) == 0 {
		return "", ErrNoCompatibleWarehouses
	}
	if len(names) == 1 {
		return lastWarehouseID, nil
	}
	return cmdio.Select(ctx, names, "Choose SQL Warehouse")
}
