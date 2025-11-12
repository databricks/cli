package databricks

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/sql"
)

// ExecuteQueryArgs represents arguments for executing a SQL query
type ExecuteQueryArgs struct {
	Query string `json:"query"`
}

// ExecuteQuery executes a SQL query and returns the results
func (c *Client) ExecuteQuery(ctx context.Context, query string) ([]map[string]any, error) {
	// Get warehouse ID from config
	if c.config.WarehouseID == "" {
		return nil, errors.New("DATABRICKS_WAREHOUSE_ID not configured")
	}

	// Execute statement
	result, err := c.workspace.StatementExecution.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		Statement:   query,
		WarehouseId: c.config.WarehouseID,
		WaitTimeout: "30s",
		Format:      sql.FormatJsonArray,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Check status
	if result.Status.State == sql.StatementStateFailed {
		errMsg := "unknown error"
		if result.Status.Error != nil && result.Status.Error.Message != "" {
			errMsg = result.Status.Error.Message
		}
		return nil, fmt.Errorf("query failed: %s", errMsg)
	}

	// Parse results
	if result.Result == nil || result.Result.DataArray == nil {
		return []map[string]any{}, nil
	}

	// Get column names
	columns := make([]string, len(result.Manifest.Schema.Columns))
	for i, col := range result.Manifest.Schema.Columns {
		columns[i] = col.Name
	}

	// Convert data array to map
	rows := make([]map[string]any, len(result.Result.DataArray))
	for i, row := range result.Result.DataArray {
		rowMap := make(map[string]any)
		for j, val := range row {
			if j < len(columns) {
				rowMap[columns[j]] = val
			}
		}
		rows[i] = rowMap
	}

	return rows, nil
}
