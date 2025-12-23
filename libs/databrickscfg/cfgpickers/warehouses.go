package cfgpickers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
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

// sortWarehousesByState sorts warehouses by state priority (running first), then alphabetically by name.
// Deleted warehouses are filtered out.
func sortWarehousesByState(all []sql.EndpointInfo) []sql.EndpointInfo {
	var warehouses []sql.EndpointInfo
	for _, wh := range all {
		if wh.State != sql.StateDeleted && wh.State != sql.StateDeleting {
			warehouses = append(warehouses, wh)
		}
	}

	priorities := map[sql.State]int{
		sql.StateRunning:  1,
		sql.StateStarting: 2,
		sql.StateStopped:  3,
		sql.StateStopping: 4,
	}
	sort.Slice(warehouses, func(i, j int) bool {
		pi, pj := priorities[warehouses[i].State], priorities[warehouses[j].State]
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(warehouses[i].Name) < strings.ToLower(warehouses[j].Name)
	})

	return warehouses
}

// GetDefaultWarehouse returns the default warehouse for the workspace.
// It tries the following in order:
// 1. The "default" warehouse via API (server-side convention, not yet fully rolled out)
// 2. The first usable warehouse sorted by state (running first)
func GetDefaultWarehouse(ctx context.Context, w *databricks.WorkspaceClient) (*sql.EndpointInfo, error) {
	// Try the "default" warehouse convention first
	// This is a new server-side feature that may not be available everywhere yet
	warehouse, err := w.Warehouses.Get(ctx, sql.GetWarehouseRequest{Id: "default"})
	if err == nil {
		return &sql.EndpointInfo{
			Id:    warehouse.Id,
			Name:  warehouse.Name,
			State: warehouse.State,
		}, nil
	}
	var apiErr *apierr.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode >= 500 {
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
func listUsableWarehouses(ctx context.Context, w *databricks.WorkspaceClient) ([]sql.EndpointInfo, error) {
	// The SDK doesn't expose skip_cannot_use parameter, so we use the raw API
	clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
	if err != nil {
		return nil, fmt.Errorf("create HTTP client config: %w", err)
	}
	apiClient := httpclient.NewApiClient(clientCfg)

	var response sql.ListWarehousesResponse
	err = apiClient.Do(ctx, "GET", "/api/2.0/sql/warehouses?skip_cannot_use=true",
		httpclient.WithResponseUnmarshal(&response))
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	return response.Warehouses, nil
}

// SelectWarehouse prompts the user to select a SQL warehouse and returns the warehouse ID.
// Warehouses are sorted by state (running first) so the default selection is the best available.
// In non-interactive mode, returns the first (best) warehouse automatically.
// The description parameter is shown before the picker (if non-empty).
func SelectWarehouse(ctx context.Context, w *databricks.WorkspaceClient, description string, filters ...warehouseFilter) (string, error) {
	all, err := w.Warehouses.ListAll(ctx, sql.ListWarehousesRequest{})
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
		return warehouses[0].Id, nil
	}

	// The first warehouse (sorted by state, then alphabetically) is the default
	defaultId := warehouses[0].Id

	// Sort by running state first, then alphabetically for display
	sort.Slice(warehouses, func(i, j int) bool {
		iRunning := warehouses[i].State == sql.StateRunning
		jRunning := warehouses[j].State == sql.StateRunning
		if iRunning != jRunning {
			return iRunning
		}
		return strings.ToLower(warehouses[i].Name) < strings.ToLower(warehouses[j].Name)
	})

	// Build options for the picker (● = running, ○ = not running)
	var items []cmdio.Tuple
	for _, warehouse := range warehouses {
		var icon string
		if warehouse.State == sql.StateRunning {
			icon = color.GreenString("●")
		} else {
			icon = color.HiBlackString("○")
		}

		// Show type info in gray
		typeInfo := strings.ToLower(string(warehouse.WarehouseType))
		if warehouse.EnableServerlessCompute {
			typeInfo = "serverless"
		}

		name := fmt.Sprintf("%s %s %s", icon, warehouse.Name, color.HiBlackString(typeInfo))
		if warehouse.Id == defaultId {
			name += color.HiBlackString(" [DEFAULT]")
		}
		items = append(items, cmdio.Tuple{Name: name, Id: warehouse.Id})
	}

	if description != "" {
		cmdio.LogString(ctx, description)
	}
	promptui.SearchPrompt = "Search: "
	warehouseId, err := cmdio.SelectOrdered(ctx, items, "warehouse\n")
	if err != nil {
		return "", err
	}

	for _, wh := range warehouses {
		if wh.Id == warehouseId {
			cmdio.LogString(ctx, fmt.Sprintf("warehouse_id: %s (%s)", warehouseId, wh.Name))
			break
		}
	}

	return warehouseId, nil
}
