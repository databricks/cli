package cfgpickers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdkclients"
	mirror "github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/apierr"
	sql "github.com/databricks/sdk-go/warehouses/v1"
)

var ErrNoCompatibleWarehouses = errors.New("no compatible warehouses")

type warehouseFilter func(sql.EndpointInfo) bool

func WithWarehouseTypes(types ...sql.WarehouseType) func(sql.EndpointInfo) bool {
	allowed := map[sql.WarehouseType]bool{}
	for _, v := range types {
		allowed[v] = true
	}
	return func(ei sql.EndpointInfo) bool {
		return allowed[ei.WarehouseType]
	}
}

func AskForWarehouse(ctx context.Context, w *sdkclients.Clients, filters ...warehouseFilter) (string, error) {
	all, err := listAllWarehouses(ctx, w)
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
		case sql.EndpointState_Running:
			state = cmdio.Green(ctx, string(warehouse.State))
		case sql.EndpointState_Stopped, sql.EndpointState_Deleted, sql.EndpointState_Stopping, sql.EndpointState_Deleting:
			state = cmdio.Red(ctx, string(warehouse.State))
		default:
			state = cmdio.Blue(ctx, string(warehouse.State))
		}
		visibleTouser := fmt.Sprintf("%s (%s %s)", utils.Deref(warehouse.Name), state, warehouse.WarehouseType)
		names[visibleTouser] = utils.Deref(warehouse.Id)
		lastWarehouseID = utils.Deref(warehouse.Id)
	}
	if len(names) == 0 {
		return "", ErrNoCompatibleWarehouses
	}
	if len(names) == 1 {
		return lastWarehouseID, nil
	}
	return cmdio.Select(ctx, names, "Choose SQL Warehouse")
}

// sortWarehousesByState sorts warehouses by state priority (running first), then alphabetically by name.
// Deleted warehouses are filtered out.
func sortWarehousesByState(all []sql.EndpointInfo) []sql.EndpointInfo {
	var warehouses []sql.EndpointInfo
	for _, wh := range all {
		if wh.State != sql.EndpointState_Deleted && wh.State != sql.EndpointState_Deleting {
			warehouses = append(warehouses, wh)
		}
	}

	priorities := map[sql.EndpointState]int{
		sql.EndpointState_Running:  1,
		sql.EndpointState_Starting: 2,
		sql.EndpointState_Stopped:  3,
		sql.EndpointState_Stopping: 4,
	}
	slices.SortFunc(warehouses, func(a, b sql.EndpointInfo) int {
		if n := cmp.Compare(priorities[a.State], priorities[b.State]); n != 0 {
			return n
		}
		return cmp.Compare(strings.ToLower(utils.Deref(a.Name)), strings.ToLower(utils.Deref(b.Name)))
	})

	return warehouses
}

// GetDefaultWarehouse returns the default warehouse for the workspace.
// It tries the following in order:
// 1. The "default" warehouse via API (server-side convention, not yet fully rolled out)
// 2. The first usable warehouse sorted by state (running first)
func GetDefaultWarehouse(ctx context.Context, w *sdkclients.Clients) (*sql.EndpointInfo, error) {
	// Try the "default" warehouse convention first
	// This is a new server-side feature that may not be available everywhere yet
	warehouse, err := w.Warehouses.GetWarehouse(ctx, sql.GetWarehouseRequest{Id: new("default")})
	if err == nil {
		return &sql.EndpointInfo{
			Id:    warehouse.Id,
			Name:  warehouse.Name,
			State: warehouse.State,
		}, nil
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || apiErr.HTTPStatusCode() >= 500 {
		return nil, fmt.Errorf("get default warehouse: %w", err)
	}

	warehouses, err := listUsableWarehouses(ctx, w)
	if err != nil {
		return nil, err
	}
	warehouses = sortWarehousesByState(warehouses)
	if len(warehouses) == 0 {
		return nil, ErrNoCompatibleWarehouses
	}
	return &warehouses[0], nil
}

// listUsableWarehouses returns warehouses the user has permission to use.
// This uses the skip_cannot_use=true parameter to filter out inaccessible warehouses.
func listUsableWarehouses(ctx context.Context, w *sdkclients.Clients) ([]sql.EndpointInfo, error) {
	// The SDK doesn't expose skip_cannot_use parameter, so we use the raw API
	apiClient, err := apiclient.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	var response mirror.ListWarehousesResponse
	err = apiClient.Do(ctx, "GET", "/api/2.0/sql/warehouses?skip_cannot_use=true", auth.WorkspaceIDHeaders(w.Config), nil, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	converted, err := mirror.ListWarehousesResponseToSDK(&response)
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	return converted.Warehouses, nil
}

// SelectWarehouse prompts the user to select a SQL warehouse and returns the warehouse ID.
// Warehouses are sorted by state (running first) so the default selection is the best available.
// In non-interactive mode, returns the first (best) warehouse automatically.
// The description parameter is shown before the picker (if non-empty).
func SelectWarehouse(ctx context.Context, w *sdkclients.Clients, description string, filters ...warehouseFilter) (string, error) {
	all, err := listAllWarehouses(ctx, w)
	if err != nil {
		return "", fmt.Errorf("list warehouses: %w", err)
	}

	warehouses := sortWarehousesByState(all)

	// Apply filters
	var filtered []sql.EndpointInfo
	for _, wh := range warehouses {
		skip := false
		for _, filter := range filters {
			if !filter(wh) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, wh)
		}
	}
	warehouses = filtered

	if len(warehouses) == 0 {
		return "", ErrNoCompatibleWarehouses
	}

	if len(warehouses) == 1 || !cmdio.IsPromptSupported(ctx) {
		return utils.Deref(warehouses[0].Id), nil
	}

	// The first warehouse (sorted by state, then alphabetically) is the default
	defaultId := utils.Deref(warehouses[0].Id)

	// Sort by running state first, then alphabetically for display
	slices.SortFunc(warehouses, func(a, b sql.EndpointInfo) int {
		aRunning := a.State == sql.EndpointState_Running
		bRunning := b.State == sql.EndpointState_Running
		if aRunning != bRunning {
			if aRunning {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(utils.Deref(a.Name)), strings.ToLower(utils.Deref(b.Name)))
	})

	// Build options for the picker (● = running, ○ = not running)
	var items []cmdio.Tuple
	for _, warehouse := range warehouses {
		var icon string
		if warehouse.State == sql.EndpointState_Running {
			icon = cmdio.Green(ctx, "●")
		} else {
			icon = cmdio.HiBlack(ctx, "○")
		}

		// Show type info in gray
		typeInfo := strings.ToLower(string(warehouse.WarehouseType))
		if utils.Deref(warehouse.EnableServerlessCompute) {
			typeInfo = "serverless"
		}

		name := fmt.Sprintf("%s %s %s", icon, utils.Deref(warehouse.Name), cmdio.HiBlack(ctx, typeInfo))
		if utils.Deref(warehouse.Id) == defaultId {
			name += cmdio.HiBlack(ctx, " [DEFAULT]")
		}
		items = append(items, cmdio.Tuple{Name: name, Id: utils.Deref(warehouse.Id)})
	}

	if description != "" {
		cmdio.LogString(ctx, description)
	}
	warehouseId, err := cmdio.SelectOrdered(ctx, items, "warehouse\n")
	if err != nil {
		return "", err
	}

	for _, wh := range warehouses {
		if utils.Deref(wh.Id) == warehouseId {
			cmdio.LogString(ctx, fmt.Sprintf("warehouse_id: %s (%s)", warehouseId, utils.Deref(wh.Name)))
			break
		}
	}

	return warehouseId, nil
}

func listAllWarehouses(ctx context.Context, w *sdkclients.Clients) ([]sql.EndpointInfo, error) {
	var all []sql.EndpointInfo
	for wh, err := range w.Warehouses.ListWarehousesIter(ctx, sql.ListWarehousesRequest{}) {
		if err != nil {
			return nil, err
		}
		all = append(all, *wh)
	}
	return all, nil
}
