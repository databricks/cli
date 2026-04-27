package aitools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	dbsql "github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var sqlIdentifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func newDiscoverSchemaCmd() *cobra.Command {
	var concurrency int

	cmd := &cobra.Command{
		Use:   "discover-schema TABLE...",
		Short: "Discover schema for one or more tables",
		Long: `Batch discover table metadata including columns, types, sample data, and null counts.

Tables must be specified in CATALOG.SCHEMA.TABLE format.

For each table, returns:
- Column names and types
- Sample data (5 rows)
- Null counts per column
- Total row count

Multiple tables are discovered in parallel against the warehouse, capped
by --concurrency (default 8). Within a single table, the sample-data and
null-counts probes also run in parallel after the column list is known.`,
		Example: `  databricks experimental aitools tools discover-schema samples.nyctaxi.trips
  databricks experimental aitools tools discover-schema catalog.schema.table1 catalog.schema.table2`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if concurrency <= 0 {
				return errInvalidBatchConcurrency
			}
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// validate table names: each part must be a safe SQL identifier
			for _, table := range args {
				if _, err := quoteTableName(table); err != nil {
					return err
				}
			}

			w := cmdctx.WorkspaceClient(ctx)

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouseID, err := middlewares.GetWarehouseID(ctx, true)
			if err != nil {
				return err
			}

			results := make([]string, len(args))
			g := new(errgroup.Group)
			g.SetLimit(concurrency)
			for i, table := range args {
				g.Go(func() error {
					result, err := discoverTable(ctx, w, warehouseID, table)
					if err != nil {
						results[i] = fmt.Sprintf("Error discovering %s: %v", table, err)
					} else {
						results[i] = result
					}
					// A failure on one table shouldn't abort the others.
					return nil
				})
			}
			_ = g.Wait()

			// format output with dividers for multiple tables
			var output string
			if len(results) == 1 {
				output = results[0]
			} else {
				divider := strings.Repeat("-", 70)
				var sb strings.Builder
				for i, result := range results {
					if i > 0 {
						sb.WriteByte('\n')
						sb.WriteString(divider)
						sb.WriteByte('\n')
					}
					fmt.Fprintf(&sb, "TABLE: %s\n%s\n", args[i], divider)
					sb.WriteString(result)
				}
				output = sb.String()
			}

			cmdio.LogString(ctx, output)
			return nil
		},
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", defaultBatchConcurrency, "Maximum in-flight SQL statements when discovering multiple tables")

	return cmd
}

func discoverTable(ctx context.Context, w *databricks.WorkspaceClient, warehouseID, table string) (string, error) {
	quoted, err := quoteTableName(table)
	if err != nil {
		return "", err
	}

	// 1. describe table - get columns and types
	describeSQL := "DESCRIBE TABLE " + quoted
	descResp, err := executeSQL(ctx, w, warehouseID, describeSQL)
	if err != nil {
		return "", fmt.Errorf("describe table: %w", err)
	}

	columns, types := parseDescribeResult(descResp)
	if len(columns) == 0 {
		return "", errors.New("no columns found")
	}

	// 2 + 3. Sample data and null counts run in parallel; both depend only on
	// the column list (already known) and not on each other.
	sampleSQL := fmt.Sprintf("SELECT * FROM %s LIMIT 5", quoted)

	nullCountExprs := make([]string, len(columns))
	for i, col := range columns {
		nullCountExprs[i] = fmt.Sprintf("SUM(CASE WHEN `%s` IS NULL THEN 1 ELSE 0 END) AS `%s_nulls`", col, col)
	}
	nullSQL := fmt.Sprintf("SELECT COUNT(*) AS total_rows, %s FROM %s",
		strings.Join(nullCountExprs, ", "), quoted)

	var sampleResp, nullResp *dbsql.StatementResponse
	var sampleErr, nullErr error

	g := new(errgroup.Group)
	g.Go(func() error {
		sampleResp, sampleErr = executeSQL(ctx, w, warehouseID, sampleSQL)
		return nil
	})
	g.Go(func() error {
		nullResp, nullErr = executeSQL(ctx, w, warehouseID, nullSQL)
		return nil
	})
	_ = g.Wait()

	// Assemble the output in the established order: columns, sample, null counts.
	var sb strings.Builder
	sb.WriteString("COLUMNS:\n")
	for i, col := range columns {
		fmt.Fprintf(&sb, "  %s: %s\n", col, types[i])
	}

	if sampleErr != nil {
		fmt.Fprintf(&sb, "\nSAMPLE DATA: Error - %v\n", sampleErr)
	} else {
		sb.WriteString("\nSAMPLE DATA:\n")
		sb.WriteString(formatTableData(sampleResp))
	}

	if nullErr != nil {
		fmt.Fprintf(&sb, "\nNULL COUNTS: Error - %v\n", nullErr)
	} else {
		sb.WriteString("\nNULL COUNTS:\n")
		sb.WriteString(formatNullCounts(nullResp, columns))
	}

	return sb.String(), nil
}

func executeSQL(ctx context.Context, w *databricks.WorkspaceClient, warehouseID, statement string) (*dbsql.StatementResponse, error) {
	resp, err := w.StatementExecution.ExecuteStatement(ctx, dbsql.ExecuteStatementRequest{
		WarehouseId:   warehouseID,
		Statement:     statement,
		WaitTimeout:   "0s",
		OnWaitTimeout: dbsql.ExecuteStatementRequestOnWaitTimeoutContinue,
	})
	if err != nil {
		return nil, fmt.Errorf("execute statement: %w", err)
	}

	pollResp, err := pollStatement(ctx, w.StatementExecution, resp)
	if err != nil {
		return nil, err
	}

	if err := checkFailedState(pollResp.Status); err != nil {
		return nil, err
	}
	return pollResp, nil
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
		fmt.Fprintf(&sb, "  Row %d:\n", i+1)
		for j, val := range row {
			colName := fmt.Sprintf("col%d", j)
			if j < len(columns) {
				colName = columns[j]
			}
			fmt.Fprintf(&sb, "    %s: %v\n", colName, val)
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
		fmt.Fprintf(&sb, "  total_rows: %v\n", row[0])
	}

	// remaining values are null counts per column
	for i, col := range columns {
		idx := i + 1
		if idx < len(row) {
			fmt.Fprintf(&sb, "  %s_nulls: %v\n", col, row[idx])
		}
	}

	return sb.String()
}

// quoteTableName validates and backtick-quotes a CATALOG.SCHEMA.TABLE identifier.
func quoteTableName(table string) (string, error) {
	parts := strings.Split(table, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid table format %q: expected CATALOG.SCHEMA.TABLE", table)
	}
	for _, part := range parts {
		if !sqlIdentifierRe.MatchString(part) {
			return "", fmt.Errorf("invalid SQL identifier %q in table name %q", part, table)
		}
	}
	return fmt.Sprintf("`%s`.`%s`.`%s`", parts[0], parts[1], parts[2]), nil
}
