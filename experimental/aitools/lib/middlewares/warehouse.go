package middlewares

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

const (
	warehouseLoadingKey = "warehouse_loading"
	warehouseErrorKey   = "warehouse_error"
)

// loadWarehouseInBackground loads the default warehouse in a background goroutine.
func loadWarehouseInBackground(ctx context.Context) {
	sess, err := session.GetSession(ctx)
	if err != nil {
		return
	}

	// Create a WaitGroup to track loading state
	var wg sync.WaitGroup
	wg.Add(1)
	sess.Set(warehouseLoadingKey, &wg)

	defer wg.Done()

	warehouse, err := getDefaultWarehouse(ctx)
	if err != nil {
		sess.Set(warehouseErrorKey, err)
		return
	}

	sess.Set("warehouse_endpoint", warehouse)
}

// GetWarehouseEndpoint returns the resolved warehouse endpoint.
// If autoStart is true and the warehouse is stopped, it will be started automatically.
func GetWarehouseEndpoint(ctx context.Context, autoStart bool) (*sql.EndpointInfo, error) {
	sess, err := session.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Wait for background loading if in progress
	if wgRaw, ok := sess.Get(warehouseLoadingKey); ok {
		wg := wgRaw.(*sync.WaitGroup)
		wg.Wait()
		sess.Delete(warehouseLoadingKey)

		// Check if there was an error during background loading
		if errRaw, ok := sess.Get(warehouseErrorKey); ok {
			sess.Delete(warehouseErrorKey)
			return nil, errRaw.(error)
		}
	}

	warehouse, ok := sess.Get("warehouse_endpoint")
	if !ok {
		// Fallback: synchronously load if background loading didn't happen
		warehouse, err = getDefaultWarehouse(ctx)
		if err != nil {
			return nil, err
		}
		sess.Set("warehouse_endpoint", warehouse)
	}

	endpoint := warehouse.(*sql.EndpointInfo)

	if autoStart && (endpoint.State == sql.StateStopped || endpoint.State == sql.StateStopping) {
		endpoint, err = startWarehouse(ctx, endpoint.Id)
		if err != nil {
			return nil, err
		}
		sess.Set("warehouse_endpoint", endpoint)
	}

	return endpoint, nil
}

// GetWarehouseID returns the resolved warehouse ID.
// If autoStart is true and the warehouse is stopped, it will be started automatically.
func GetWarehouseID(ctx context.Context, autoStart bool) (string, error) {
	warehouse, err := GetWarehouseEndpoint(ctx, autoStart)
	if err != nil {
		return "", err
	}
	return warehouse.Id, nil
}

func startWarehouse(ctx context.Context, id string) (*sql.EndpointInfo, error) {
	w, err := GetDatabricksClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get databricks client: %w", err)
	}
	wait, err := w.Warehouses.Start(ctx, sql.StartRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("start warehouse %s: %w", id, err)
	}
	resp, err := wait.Get()
	if err != nil {
		return nil, fmt.Errorf("wait for warehouse %s to start: %w", id, err)
	}
	return &sql.EndpointInfo{
		Id:    resp.Id,
		Name:  resp.Name,
		State: resp.State,
	}, nil
}

func getDefaultWarehouse(ctx context.Context) (*sql.EndpointInfo, error) {
	w, err := GetDatabricksClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get databricks client: %w", err)
	}
	return resolveWarehouse(ctx, w)
}

// resolveWarehouse selects a warehouse using the following priority:
// 1. DATABRICKS_WAREHOUSE_ID env var
// 2. User's default warehouse override (CUSTOM type only)
// 3. Server-side default / first usable warehouse by state
func resolveWarehouse(ctx context.Context, w *databricks.WorkspaceClient) (*sql.EndpointInfo, error) {
	// first resolve DATABRICKS_WAREHOUSE_ID env variable
	warehouseID := env.Get(ctx, "DATABRICKS_WAREHOUSE_ID")
	if warehouseID != "" {
		warehouse, err := w.Warehouses.Get(ctx, sql.GetWarehouseRequest{
			Id: warehouseID,
		})
		if err != nil {
			return nil, fmt.Errorf("get warehouse: %w", err)
		}
		return &sql.EndpointInfo{
			Id:    warehouse.Id,
			Name:  warehouse.Name,
			State: warehouse.State,
		}, nil
	}

	// Check user's default warehouse override (set via the SQL UI or CLI).
	// Only CUSTOM overrides are used; LAST_SELECTED requires UI state we don't have.
	override, err := w.Warehouses.GetDefaultWarehouseOverride(ctx, sql.GetDefaultWarehouseOverrideRequest{
		Name: "default-warehouse-overrides/me",
	})
	if err == nil && override.Type == sql.DefaultWarehouseOverrideTypeCustom && override.WarehouseId != "" {
		warehouse, err := w.Warehouses.Get(ctx, sql.GetWarehouseRequest{
			Id: override.WarehouseId,
		})
		if err == nil && warehouse.State != sql.StateDeleted && warehouse.State != sql.StateDeleting {
			return &sql.EndpointInfo{
				Id:    warehouse.Id,
				Name:  warehouse.Name,
				State: warehouse.State,
			}, nil
		}
	}

	return cfgpickers.GetDefaultWarehouse(ctx, w)
}
