package middlewares

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/env"
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

func GetWarehouseEndpoint(ctx context.Context) (*sql.EndpointInfo, error) {
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

	return warehouse.(*sql.EndpointInfo), nil
}

func GetWarehouseID(ctx context.Context) (string, error) {
	warehouse, err := GetWarehouseEndpoint(ctx)
	if err != nil {
		return "", err
	}
	return warehouse.Id, nil
}

func getDefaultWarehouse(ctx context.Context) (*sql.EndpointInfo, error) {
	w, err := GetDatabricksClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get databricks client: %w", err)
	}

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

	return cfgpickers.GetDefaultWarehouse(ctx, w)
}
