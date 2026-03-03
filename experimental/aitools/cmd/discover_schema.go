package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	dbsql "github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func newDiscoverSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover-schema TABLE...",
		Short: "Discover schema for one or more tables",
		Long: `Batch discover table metadata including columns, types, sample data, and null counts.

Tables must be specified in CATALOG.SCHEMA.TABLE format.

For each table, returns:
- Column names and types
- Sample data (5 rows)
- Null counts per column
- Total row count`,
		Example: `  databricks experimental aitools discover-schema samples.nyctaxi.trips
  databricks experimental aitools discover-schema catalog.schema.table1 catalog.schema.table2`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			// validate table names
			for _, table := range args {
				parts := strings.Split(table, ".")
				if len(parts) != 3 {
					return fmt.Errorf("invalid table format %q: expected CATALOG.SCHEMA.TABLE", table)
				}
			}

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouseID, err := middlewares.GetWarehouseID(ctx, true)
			if err != nil {
				return err
			}

			var results []string
			for _, table := range args {
				result, err := discoverTable(ctx, w, warehouseID, table)
				if err != nil {
					result = fmt.Sprintf("Error discovering %s: %v", table, err)
				}
				results = append(results, result)
			}

			// format output with dividers for multiple tables
			var output string
			if len(results) == 1 {
				output = results[0]
			} else {
				divider := strings.Repeat("-", 70)
				for i, result := range results {
					if i > 0 {
						output += "\n" + divider + "\n"
					}
					output += fmt.Sprintf("TABLE: %s\n%s\n", args[i], divider)
					output += result
				}
			}

			cmdio.LogString(ctx, output)
			return nil
		},
	}

	return cmd
}

func discoverTable(ctx context.Context, w *databricks.WorkspaceClient, warehouseID, table string) (string, error) {
	var sb strings.Builder

	// 1. describe table - get columns and types
	describeSQL := "DESCRIBE TABLE " + table
	descResp, err := executeSQL(ctx, w, warehouseID, describeSQL)
	if err != nil {
		return "", fmt.Errorf("describe table: %w", err)
	}

	columns, types := parseDescribeResult(descResp)
	if len(columns) == 0 {
		return "", errors.New("no columns found")
	}

	sb.WriteString("COLUMNS:\n")
	for i, col := range columns {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", col, types[i]))
	}

	// 2. sample data (5 rows)
	sampleSQL := fmt.Sprintf("SELECT * FROM %s LIMIT 5", table)
	sampleResp, err := executeSQL(ctx, w, warehouseID, sampleSQL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("\nSAMPLE DATA: Error - %v\n", err))
	} else {
		sb.WriteString("\nSAMPLE DATA:\n")
		sb.WriteString(formatTableData(sampleResp))
	}

	// 3. null counts per column
	nullCountExprs := make([]string, len(columns))
	for i, col := range columns {
		nullCountExprs[i] = fmt.Sprintf("SUM(CASE WHEN `%s` IS NULL THEN 1 ELSE 0 END) AS `%s_nulls`", col, col)
	}
	nullSQL := fmt.Sprintf("SELECT COUNT(*) AS total_rows, %s FROM %s",
		strings.Join(nullCountExprs, ", "), table)

	nullResp, err := executeSQL(ctx, w, warehouseID, nullSQL)
	if err != nil {
		sb.WriteString(fmt.Sprintf("\nNULL COUNTS: Error - %v\n", err))
	} else {
		sb.WriteString("\nNULL COUNTS:\n")
		sb.WriteString(formatNullCounts(nullResp, columns))
	}

	return sb.String(), nil
}

func executeSQL(ctx context.Context, w *databricks.WorkspaceClient, warehouseID, statement string) (*dbsql.StatementResponse, error) {
	resp, err := w.StatementExecution.ExecuteAndWait(ctx, dbsql.ExecuteStatementRequest{
		WarehouseId: warehouseID,
		Statement:   statement,
		WaitTimeout: "50s",
	})
	if err != nil {
		return nil, err
	}

	if resp.Status != nil && resp.Status.State == dbsql.StatementStateFailed {
		errMsg := "query failed"
		if resp.Status.Error != nil {
			errMsg = resp.Status.Error.Message
		}
		return nil, errors.New(errMsg)
	}

	return resp, nil
}

func parseDescribeResult(resp *dbsql.StatementResponse) (columns, types []string) {
	if resp.Result == nil || resp.Result.DataArray == nil {
		return nil, nil
	}

	for _, row := range resp.Result.DataArray {
		if len(row) < 2 {
			continue
		}
		colName := row[0]
		colType := row[1]
		// skip partition/metadata rows (they start with #)
		if strings.HasPrefix(colName, "#") || colName == "" {
			continue
		}
		columns = append(columns, colName)
		types = append(types, colType)
	}
	return columns, types
}

func formatTableData(resp *dbsql.StatementResponse) string {
	if resp.Result == nil || resp.Result.DataArray == nil || len(resp.Result.DataArray) == 0 {
		return "  (no data)\n"
	}

	var sb strings.Builder
	var columns []string
	if resp.Manifest != nil && resp.Manifest.Schema != nil {
		for _, col := range resp.Manifest.Schema.Columns {
			columns = append(columns, col.Name)
		}
	}

	for i, row := range resp.Result.DataArray {
		sb.WriteString(fmt.Sprintf("  Row %d:\n", i+1))
		for j, val := range row {
			colName := fmt.Sprintf("col%d", j)
			if j < len(columns) {
				colName = columns[j]
			}
			sb.WriteString(fmt.Sprintf("    %s: %v\n", colName, val))
		}
	}
	return sb.String()
}

func formatNullCounts(resp *dbsql.StatementResponse, columns []string) string {
	if resp.Result == nil || resp.Result.DataArray == nil || len(resp.Result.DataArray) == 0 {
		return "  (no data)\n"
	}

	row := resp.Result.DataArray[0]
	var sb strings.Builder

	// first value is total_rows
	if len(row) > 0 {
		sb.WriteString(fmt.Sprintf("  total_rows: %v\n", row[0]))
	}

	// remaining values are null counts per column
	for i, col := range columns {
		idx := i + 1
		if idx < len(row) {
			sb.WriteString(fmt.Sprintf("  %s_nulls: %v\n", col, row[idx]))
		}
	}

	return sb.String()
}
