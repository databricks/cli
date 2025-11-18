package databricks

import (
	"context"
	"errors"
	"fmt"
	"time"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// ExecuteQueryArgs represents arguments for executing a SQL query
type ExecuteQueryArgs struct {
	Query       string  `json:"query"`
	WarehouseID *string `json:"warehouse_id,omitempty"`
	MaxRows     *int    `json:"max_rows,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"`
}

// ExecuteQueryResult represents the result of a SQL query execution
type ExecuteQueryResult struct {
	Columns       []string `json:"columns"`
	Rows          [][]any  `json:"rows"`
	RowCount      int      `json:"row_count"`
	Truncated     bool     `json:"truncated"`
	ExecutionTime float64  `json:"execution_time_seconds"`
}

// ExecuteQuery executes a SQL query and returns the results
func ExecuteQuery(ctx context.Context, cfg *mcp.Config, args *ExecuteQueryArgs) (*ExecuteQueryResult, error) {
	// Determine warehouse ID
	warehouseID := cfg.WarehouseID
	if args.WarehouseID != nil {
		warehouseID = *args.WarehouseID
	}
	if warehouseID == "" {
		return nil, errors.New("DATABRICKS_WAREHOUSE_ID not configured")
	}

	// Set max rows with default and limit
	maxRows := 1000
	if args.MaxRows != nil {
		maxRows = min(*args.MaxRows, 10000)
	}

	// Set timeout with default
	timeout := 60 * time.Second
	if args.Timeout != nil {
		timeout = time.Duration(*args.Timeout) * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	w := cmdctx.WorkspaceClient(ctx)

	// Calculate wait timeout for the query (slightly less than context timeout)
	waitTimeout := fmt.Sprintf("%ds", int(timeout.Seconds())-5)
	if timeout < 10*time.Second {
		waitTimeout = fmt.Sprintf("%ds", max(int(timeout.Seconds())-1, 1))
	}

	// Execute statement
	result, err := w.StatementExecution.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		Statement:   args.Query,
		WarehouseId: warehouseID,
		WaitTimeout: waitTimeout,
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

	executionTime := time.Since(startTime).Seconds()

	// Parse results
	if result.Result == nil || result.Result.DataArray == nil {
		return &ExecuteQueryResult{
			Columns:       []string{},
			Rows:          [][]any{},
			RowCount:      0,
			Truncated:     false,
			ExecutionTime: executionTime,
		}, nil
	}

	// Get column names
	columns := make([]string, len(result.Manifest.Schema.Columns))
	for i, col := range result.Manifest.Schema.Columns {
		columns[i] = col.Name
	}

	// Convert data array and apply row limit
	totalRows := len(result.Result.DataArray)
	truncated := totalRows > maxRows
	rowCount := min(totalRows, maxRows)

	rows := make([][]any, rowCount)
	for i := range rowCount {
		sourceRow := result.Result.DataArray[i]
		rows[i] = make([]any, len(sourceRow))
		for j, val := range sourceRow {
			rows[i][j] = val
		}
	}

	return &ExecuteQueryResult{
		Columns:       columns,
		Rows:          rows,
		RowCount:      rowCount,
		Truncated:     truncated,
		ExecutionTime: executionTime,
	}, nil
}
