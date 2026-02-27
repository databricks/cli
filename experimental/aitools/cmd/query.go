package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

const (
	// pollIntervalInitial is the starting interval between status polls.
	pollIntervalInitial = 1 * time.Second

	// pollIntervalMax is the maximum interval between status polls.
	pollIntervalMax = 5 * time.Second

	// cancelTimeout is how long to wait for server-side cancellation.
	cancelTimeout = 10 * time.Second
)

func newQueryCmd() *cobra.Command {
	var warehouseID string

	cmd := &cobra.Command{
		Use:   "query SQL",
		Short: "Execute SQL against a Databricks warehouse",
		Long: `Execute a SQL statement against a Databricks SQL warehouse and return results.

The command auto-detects an available warehouse unless --warehouse is set
or the DATABRICKS_WAREHOUSE_ID environment variable is configured.

Output includes the query results as JSON and row count.`,
		Example: `  databricks experimental aitools tools query "SELECT * FROM samples.nyctaxi.trips LIMIT 5"
  databricks experimental aitools tools query --warehouse abc123 "SELECT 1"`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			sqlStatement := cleanSQL(args[0])
			if sqlStatement == "" {
				return errors.New("SQL statement is required")
			}

			wID, err := resolveWarehouseID(ctx, w, warehouseID)
			if err != nil {
				return err
			}

			resp, err := executeAndPoll(ctx, w.StatementExecution, wID, sqlStatement)
			if err != nil {
				return err
			}

			output, err := formatQueryResult(resp)
			if err != nil {
				return err
			}

			cmdio.LogString(ctx, output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&warehouseID, "warehouse", "w", "", "SQL warehouse ID to use for execution")

	return cmd
}

// resolveWarehouseID returns the warehouse ID to use for query execution.
// Priority: explicit flag > middleware auto-detection (env var > server default > first running).
func resolveWarehouseID(ctx context.Context, w any, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	sess := session.NewSession()
	sess.Set(middlewares.DatabricksClientKey, w)
	ctx = session.WithSession(ctx, sess)

	return middlewares.GetWarehouseID(ctx, true)
}

// executeAndPoll submits a SQL statement asynchronously and polls until completion.
// It shows a spinner in interactive mode and supports Ctrl+C cancellation.
func executeAndPoll(ctx context.Context, api sql.StatementExecutionInterface, warehouseID, statement string) (*sql.StatementResponse, error) {
	// Submit asynchronously to get the statement ID immediately for cancellation.
	resp, err := api.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		WarehouseId: warehouseID,
		Statement:   statement,
		WaitTimeout: "0s",
	})
	if err != nil {
		return nil, fmt.Errorf("execute statement: %w", err)
	}

	statementID := resp.StatementId

	// Check if it completed immediately.
	if isTerminalState(resp.Status) {
		return resp, checkFailedState(resp.Status)
	}

	// Set up Ctrl+C: signal cancels the poll context, cleanup is unified below.
	pollCtx, pollCancel := context.WithCancel(ctx)
	defer pollCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			log.Infof(ctx, "Received interrupt, cancelling query %s", statementID)
			pollCancel()
		case <-pollCtx.Done():
		}
	}()

	// cancelStatement performs best-effort server-side cancellation.
	// Called on any poll exit due to context cancellation (signal or parent).
	cancelStatement := func() {
		cancelCtx, cancel := context.WithTimeout(context.Background(), cancelTimeout)
		defer cancel()
		if err := api.CancelExecution(cancelCtx, sql.CancelExecutionRequest{
			StatementId: statementID,
		}); err != nil {
			log.Warnf(ctx, "Failed to cancel statement %s: %v", statementID, err)
		}
	}

	// Spinner for interactive feedback, updated every second via ticker.
	sp := cmdio.NewSpinner(pollCtx)
	defer sp.Close()
	start := time.Now()
	sp.Update("Executing query...")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(start).Truncate(time.Second)
				sp.Update(fmt.Sprintf("Executing query... (%s elapsed)", elapsed))
			}
		}
	}()

	// Poll with additive backoff: 1s, 2s, 3s, 4s, 5s (capped).
	interval := pollIntervalInitial
	for {
		select {
		case <-pollCtx.Done():
			cancelStatement()
			cmdio.LogString(ctx, "Query cancelled.")
			return nil, root.ErrAlreadyPrinted
		case <-time.After(interval):
		}

		log.Debugf(ctx, "Polling statement %s: %s elapsed", statementID, time.Since(start).Truncate(time.Second))

		pollResp, err := api.GetStatementByStatementId(pollCtx, statementID)
		if err != nil {
			if pollCtx.Err() != nil {
				cancelStatement()
				cmdio.LogString(ctx, "Query cancelled.")
				return nil, root.ErrAlreadyPrinted
			}
			return nil, fmt.Errorf("poll statement status: %w", err)
		}

		if isTerminalState(pollResp.Status) {
			sp.Close()
			if err := checkFailedState(pollResp.Status); err != nil {
				return nil, err
			}
			return &sql.StatementResponse{
				StatementId: pollResp.StatementId,
				Status:      pollResp.Status,
				Manifest:    pollResp.Manifest,
				Result:      pollResp.Result,
			}, nil
		}

		interval = min(interval+time.Second, pollIntervalMax)
	}
}

// isTerminalState returns true if the statement has reached a final state.
func isTerminalState(status *sql.StatementStatus) bool {
	if status == nil {
		return false
	}
	switch status.State {
	case sql.StatementStateSucceeded, sql.StatementStateFailed,
		sql.StatementStateCanceled, sql.StatementStateClosed:
		return true
	case sql.StatementStatePending, sql.StatementStateRunning:
		return false
	}
	return false
}

// checkFailedState returns an error if the statement is in a non-success terminal state.
func checkFailedState(status *sql.StatementStatus) error {
	if status == nil {
		return nil
	}
	switch status.State {
	case sql.StatementStateFailed:
		msg := "query failed"
		if status.Error != nil {
			msg = fmt.Sprintf("query failed: %s %s", status.Error.ErrorCode, status.Error.Message)
			if strings.Contains(status.Error.Message, "UNRESOLVED_MAP_KEY") {
				msg += "\n\nHint: your shell may have stripped quotes from the SQL string. " +
					"Use single quotes for map keys (e.g. info['key']) or pass the query via --file."
			}
		}
		return errors.New(msg)
	case sql.StatementStateCanceled:
		return errors.New("query was cancelled")
	case sql.StatementStateClosed:
		return errors.New("query was closed before results could be fetched")
	case sql.StatementStatePending, sql.StatementStateRunning, sql.StatementStateSucceeded:
		return nil
	}
	return nil
}

// cleanSQL removes surrounding quotes, empty lines, and SQL comments.
func cleanSQL(s string) string {
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		s = s[1 : len(s)-1]
	}

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func formatQueryResult(resp *sql.StatementResponse) (string, error) {
	var sb strings.Builder

	if resp.Manifest == nil || resp.Result == nil {
		sb.WriteString("Query executed successfully (no results)\n")
		return sb.String(), nil
	}

	var columns []string
	if resp.Manifest.Schema != nil {
		for _, col := range resp.Manifest.Schema.Columns {
			columns = append(columns, col.Name)
		}
	}

	var rows []map[string]any
	if resp.Result.DataArray != nil {
		for _, row := range resp.Result.DataArray {
			rowMap := make(map[string]any)
			for i, val := range row {
				if i < len(columns) {
					rowMap[columns[i]] = val
				}
			}
			rows = append(rows, rowMap)
		}
	}

	output, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}

	sb.Write(output)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Row count: %d\n", len(rows)))

	return sb.String(), nil
}
