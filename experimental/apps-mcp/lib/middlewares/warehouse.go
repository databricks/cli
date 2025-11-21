package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"

	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func GetWarehouseID(ctx context.Context) (string, error) {
	sess, err := session.GetSession(ctx)
	if err != nil {
		return "", err
	}
	warehouseID, ok := sess.Get("warehouse_id")
	if !ok {
		warehouse, err := getDefaultWarehouse(ctx)
		if err != nil {
			return "", err
		}
		warehouseID = warehouse.Id
		sess.Set("warehouse_id", warehouseID.(string))
	}

	return warehouseID.(string), nil
}

func getDefaultWarehouse(ctx context.Context) (*sql.EndpointInfo, error) {
	// first resolve DATABRICKS_WAREHOUSE_ID env variable
	warehouseID := env.Get(ctx, "DATABRICKS_WAREHOUSE_ID")
	if warehouseID != "" {
		w := MustGetDatabricksClient(ctx)
		warehouse, err := w.Warehouses.Get(ctx, sql.GetWarehouseRequest{
			Id: warehouseID,
		})
		if err != nil {
			return nil, fmt.Errorf("get warehouse: %w", err)
		}
		return &sql.EndpointInfo{
			Id: warehouse.Id,
		}, nil
	}

	apiClient, err := MustGetApiClient(ctx)
	if err != nil {
		return nil, err
	}

	apiPath := "/api/2.0/sql/warehouses"
	params := url.Values{}
	params.Add("skip_cannot_use", "true")
	fullPath := fmt.Sprintf("%s?%s", apiPath, params.Encode())

	var response sql.ListWarehousesResponse
	err = apiClient.Do(ctx, "GET", fullPath, httpclient.WithResponseUnmarshal(&response))
	if err != nil {
		return nil, err
	}

	priorities := map[sql.State]int{
		sql.StateRunning:  1,
		sql.StateStarting: 2,
		sql.StateStopped:  3,
		sql.StateStopping: 4,
		sql.StateDeleted:  99,
		sql.StateDeleting: 99,
	}

	warehouses := response.Warehouses
	sort.Slice(warehouses, func(i, j int) bool {
		return priorities[warehouses[i].State] < priorities[warehouses[j].State]
	})

	if len(warehouses) == 0 {
		return nil, errNoWarehouses()
	}

	firstWarehouse := warehouses[0]
	if firstWarehouse.State == sql.StateDeleted || firstWarehouse.State == sql.StateDeleting {
		return nil, errNoWarehouses()
	}

	return &firstWarehouse, nil
}

func errNoWarehouses() error {
	return errors.New("no warehouse found. You can explicitly set the warehouse ID using the DATABRICKS_WAREHOUSE_ID environment variable")
}
