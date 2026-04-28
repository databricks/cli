package aitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	dbsql "github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var sqlIdentifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// sqlGate caps in-flight SQL statements globally and records each statement_id
// so a Ctrl+C sweep can cancel anything still running server-side. The gate's
// concurrency limit applies across all probes (DESCRIBE, sample SELECT, null
// counts) and across all tables, so --concurrency really means "max statements
// in flight," not "max tables in flight."
type sqlGate struct {
	sem chan struct{}
	mu  sync.Mutex
	ids []string
}

func newSQLGate(limit int) *sqlGate {
	return &sqlGate{sem: make(chan struct{}, limit)}
}

// run executes a SQL statement asynchronously, polls until terminal, and
// records the statement_id so it can be cancelled if the parent context is
// cancelled. Acquires a slot from the gate before submitting and releases it
// when polling completes (or the caller's context is cancelled).
func (g *sqlGate) run(ctx context.Context, w *databricks.WorkspaceClient, warehouseID, statement string) (*dbsql.StatementResponse, error) {
	// If the caller cancelled before we even tried, don't enter the select:
	// when the gate has free slots both cases are ready and Go picks one
	// pseudo-randomly. Without this early-out we'd occasionally submit a
	// statement under a cancelled context.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case g.sem <- struct{}{}:
		defer func() { <-g.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	resp, err := w.StatementExecution.ExecuteStatement(ctx, dbsql.ExecuteStatementRequest{
		WarehouseId:   warehouseID,
		Statement:     statement,
		WaitTimeout:   "0s",
		OnWaitTimeout: dbsql.ExecuteStatementRequestOnWaitTimeoutContinue,
	})
	if err != nil {
		return nil, fmt.Errorf("execute statement: %w", err)
	}

	g.mu.Lock()
	g.ids = append(g.ids, resp.StatementId)
	g.mu.Unlock()

	pollResp, err := pollStatement(ctx, w.StatementExecution, resp)
	if err != nil {
		return nil, err
	}
	if err := checkFailedState(pollResp.Status); err != nil {
		return nil, err
	}
	return pollResp, nil
}

// trackedIDs returns a snapshot of statement_ids submitted through this gate.
func (g *sqlGate) trackedIDs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.ids)
}

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

Tables and probes (DESCRIBE, sample SELECT, null counts) all share a
single warehouse-statement budget. --concurrency (default 8) caps the
total number of statements in flight at any moment, regardless of how
many tables you pass in.

On Ctrl+C, in-flight statements are cancelled server-side via
CancelExecution before the command exits.`,
		Example: `  databricks experimental aitools tools discover-schema samples.nyctaxi.trips
  databricks experimental aitools tools discover-schema catalog.schema.table1 catalog.schema.table2`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if concurrency <= 0 {
				return errInvalidBatchConcurrency
			}
			// Reject malformed identifiers before any auth/profile work.
			for _, table := range args {
				if _, err := quoteTableName(table); err != nil {
					return err
				}
			}
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			// set up session with client for middleware compatibility
			sess := session.NewSession()
			sess.Set(middlewares.DatabricksClientKey, w)
			ctx = session.WithSession(ctx, sess)

			warehouseID, err := middlewares.GetWarehouseID(ctx, true)
			if err != nil {
				return err
			}

			pollCtx, pollCancel := context.WithCancel(ctx)
			defer pollCancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigCh)

			go func() {
				select {
				case <-sigCh:
					log.Infof(ctx, "Received interrupt, cancelling in-flight discover-schema statements")
					pollCancel()
				case <-pollCtx.Done():
				}
			}()

			gate := newSQLGate(concurrency)

			results := make([]string, len(args))
			g := new(errgroup.Group)
			for i, table := range args {
				g.Go(func() error {
					result, err := discoverTable(pollCtx, gate, w, warehouseID, table)
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

			if pollCtx.Err() != nil {
				cancelDiscoverInFlight(ctx, w.StatementExecution, gate.trackedIDs())
				return root.ErrAlreadyPrinted
			}

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

	cmd.Flags().IntVar(&concurrency, "concurrency", defaultBatchConcurrency, "Maximum SQL statements in flight at once across all tables and probes")

	return cmd
}

// cancelDiscoverInFlight sends CancelExecution for every recorded statement_id.
// Best effort: errors are logged but don't fail the user-visible exit.
// Statements that already finished server-side return an error which we just
// swallow at warn level; the alternative (per-statement state tracking) isn't
// worth the bookkeeping here.
func cancelDiscoverInFlight(ctx context.Context, api dbsql.StatementExecutionInterface, ids []string) {
	if len(ids) == 0 {
		cmdio.LogString(ctx, "discover-schema cancelled.")
		return
	}
	for _, id := range ids {
		cancelCtx, cancel := context.WithTimeout(ctx, cancelTimeout)
		if err := api.CancelExecution(cancelCtx, dbsql.CancelExecutionRequest{StatementId: id}); err != nil {
			log.Warnf(ctx, "Failed to cancel statement %s: %v", id, err)
		}
		cancel()
	}
	cmdio.LogString(ctx, fmt.Sprintf("discover-schema cancelled; sent CancelExecution for %d statement(s).", len(ids)))
}

func discoverTable(ctx context.Context, gate *sqlGate, w *databricks.WorkspaceClient, warehouseID, table string) (string, error) {
	quoted, err := quoteTableName(table)
	if err != nil {
		return "", err
	}

	// 1. describe table - get columns and types
	descResp, err := gate.run(ctx, w, warehouseID, "DESCRIBE TABLE "+quoted)
	if err != nil {
		return "", fmt.Errorf("describe table: %w", err)
	}

	columns, types := parseDescribeResult(descResp)
	if len(columns) == 0 {
		return "", errors.New("no columns found")
	}

	// 2 + 3. Sample data and null counts run in parallel; both depend only on
	// the column list (already known) and not on each other. The gate (not
	// errgroup) is what actually limits warehouse concurrency.
	sampleSQL := fmt.Sprintf("SELECT * FROM %s LIMIT 5", quoted)

	nullCountExprs := make([]string, len(columns))
	for i, col := range columns {
		// Backticks inside an identifier are escaped by doubling them in
		// Databricks/Delta SQL (`` ` `` → `` `` ``). Without this, a column
		// name containing a backtick would terminate the quoted identifier
		// mid-string and produce a PARSE_SYNTAX_ERROR. Sample-data uses
		// SELECT * so the failure shows up only as a confusing
		// "NULL COUNTS: Error - ..." line in the user-facing output.
		escaped := strings.ReplaceAll(col, "`", "``")
		nullCountExprs[i] = fmt.Sprintf("SUM(CASE WHEN `%s` IS NULL THEN 1 ELSE 0 END) AS `%s_nulls`", escaped, escaped)
	}
	nullSQL := fmt.Sprintf("SELECT COUNT(*) AS total_rows, %s FROM %s",
		strings.Join(nullCountExprs, ", "), quoted)

	var sampleResp, nullResp *dbsql.StatementResponse
	var sampleErr, nullErr error

	g := new(errgroup.Group)
	g.Go(func() error {
		sampleResp, sampleErr = gate.run(ctx, w, warehouseID, sampleSQL)
		return nil
	})
	g.Go(func() error {
		nullResp, nullErr = gate.run(ctx, w, warehouseID, nullSQL)
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
