package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

const (
	// sqlFileExtension is the file extension used to auto-detect SQL files.
	sqlFileExtension = ".sql"

	// pollIntervalInitial is the starting interval between status polls.
	pollIntervalInitial = 1 * time.Second

	// pollIntervalMax is the maximum interval between status polls.
	pollIntervalMax = 5 * time.Second

	// cancelTimeout is how long to wait for server-side cancellation.
	cancelTimeout = 10 * time.Second

	// staticTableThreshold is the maximum number of rows rendered as a static table.
	// Beyond this, an interactive scrollable table is used.
	staticTableThreshold = 30

	// outputCSV is the csv output format, supported only by the query command.
	outputCSV = "csv"

	// envOutputFormat matches the env var name in cmd/root/io.go.
	envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"
)

type queryOutputMode int

const (
	queryOutputModeJSON queryOutputMode = iota
	queryOutputModeStaticTable
	queryOutputModeInteractiveTable
)

func selectQueryOutputMode(outputType flags.Output, stdoutInteractive, promptSupported bool, rowCount int) queryOutputMode {
	if outputType == flags.OutputJSON {
		return queryOutputModeJSON
	}
	if !stdoutInteractive {
		return queryOutputModeJSON
	}
	// Interactive table browsing requires keyboard input from stdin.
	// If prompts are not supported, prefer static table output instead.
	if !promptSupported {
		return queryOutputModeStaticTable
	}
	if rowCount <= staticTableThreshold {
		return queryOutputModeStaticTable
	}
	return queryOutputModeInteractiveTable
}

func newQueryCmd() *cobra.Command {
	var warehouseID string
	var filePath string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "query [SQL | file.sql]",
		Short: "Execute SQL against a Databricks warehouse",
		Long: `Execute a SQL statement against a Databricks SQL warehouse and return results.

SQL can be provided as a positional argument, read from a file with --file,
or piped via stdin. If the positional argument ends in .sql and the file
exists, it is read as a SQL file automatically.

The command auto-detects an available warehouse unless --warehouse is set
or the DATABRICKS_WAREHOUSE_ID environment variable is configured.

Output is JSON in non-interactive contexts. In interactive terminals it renders
tables, and large results open an interactive table browser. Use --output csv
to export results as CSV.`,
		Example: `  databricks experimental aitools tools query "SELECT * FROM samples.nyctaxi.trips LIMIT 5"
  databricks experimental aitools tools query --warehouse abc123 "SELECT 1"
  databricks experimental aitools tools query --file report.sql
  databricks experimental aitools tools query report.sql
  databricks experimental aitools tools query --output csv "SELECT * FROM samples.nyctaxi.trips LIMIT 5"
  echo "SELECT 1" | databricks experimental aitools tools query`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Normalize case to match root --output behavior (flags.Output.Set lowercases).
			outputFormat = strings.ToLower(outputFormat)

			// If --output wasn't explicitly passed, check the env var.
			// Invalid env values are silently ignored, matching cmd/root/io.go.
			if !cmd.Flag("output").Changed {
				if v, ok := env.Lookup(ctx, envOutputFormat); ok {
					switch flags.Output(strings.ToLower(v)) {
					case flags.OutputText, flags.OutputJSON, outputCSV:
						outputFormat = strings.ToLower(v)
					}
				}
			}

			switch flags.Output(outputFormat) {
			case flags.OutputText, flags.OutputJSON, outputCSV:
			default:
				return fmt.Errorf("unsupported output format %q, accepted values: text, json, csv", outputFormat)
			}

			w := cmdctx.WorkspaceClient(ctx)

			sqlStatement, err := resolveSQL(ctx, cmd, args, filePath)
			if err != nil {
				return err
			}

			wID, err := resolveWarehouseID(ctx, w, warehouseID)
			if err != nil {
				return err
			}

			resp, err := executeAndPoll(ctx, w.StatementExecution, wID, sqlStatement)
			if err != nil {
				return err
			}

			columns := extractColumns(resp.Manifest)
			rows, err := fetchAllRows(ctx, w.StatementExecution, resp)
			if err != nil {
				return err
			}

			// CSV bypasses the normal output mode selection.
			if flags.Output(outputFormat) == outputCSV {
				if len(columns) == 0 && len(rows) == 0 {
					return nil
				}
				return renderCSV(cmd.OutOrStdout(), columns, rows)
			}

			if len(columns) == 0 && len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Query executed successfully (no results)")
				return nil
			}

			// Output format depends on stdout capabilities.
			// Interactive table browsing also requires prompt-capable stdin.
			stdoutInteractive := cmdio.SupportsColor(ctx, cmd.OutOrStdout())
			promptSupported := cmdio.IsPromptSupported(ctx)

			switch selectQueryOutputMode(flags.Output(outputFormat), stdoutInteractive, promptSupported, len(rows)) {
			case queryOutputModeJSON:
				return renderJSON(cmd.OutOrStdout(), columns, rows)
			case queryOutputModeStaticTable:
				return renderStaticTable(cmd.OutOrStdout(), columns, rows)
			default:
				return renderInteractiveTable(cmd.OutOrStdout(), columns, rows)
			}
		},
	}

	cmd.Flags().StringVarP(&warehouseID, "warehouse", "w", "", "SQL warehouse ID to use for execution")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to a SQL file to execute")
	// Local --output flag shadows the root command's persistent --output flag,
	// adding csv support for this command only.
	cmd.Flags().StringVarP(&outputFormat, "output", "o", string(flags.OutputText), "Output format: text, json, or csv")
	cmd.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{string(flags.OutputText), string(flags.OutputJSON), string(outputCSV)}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// resolveSQL determines the SQL statement to execute from the available input sources.
// Priority: --file flag > positional arg > stdin.
func resolveSQL(ctx context.Context, cmd *cobra.Command, args []string, filePath string) (string, error) {
	var raw string

	switch {
	case filePath != "":
		if len(args) > 0 {
			return "", errors.New("cannot use both --file and a positional SQL argument")
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read SQL file: %w", err)
		}
		raw = string(data)

	case len(args) > 0:
		// If the argument looks like a .sql file, try to read it.
		// Only fall through to literal SQL if the file doesn't exist.
		// Surface other errors (permission denied, etc.) directly.
		if strings.HasSuffix(args[0], sqlFileExtension) {
			data, err := os.ReadFile(args[0])
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("read SQL file: %w", err)
			}
			if err == nil {
				raw = string(data)
				break
			}
		}
		raw = args[0]

	default:
		// No args: try reading from stdin if it's piped.
		// If stdin was overridden (e.g. cmd.SetIn in tests), always read from it.
		// Otherwise, only read if stdin is not a TTY (i.e. piped input).
		in := cmd.InOrStdin()
		_, isOsFile := in.(*os.File)
		if isOsFile && cmdio.IsPromptSupported(ctx) {
			return "", errors.New("no SQL provided; pass a SQL string, use --file, or pipe via stdin")
		}
		data, err := io.ReadAll(in)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		raw = string(data)
	}

	result := cleanSQL(raw)
	if result == "" {
		return "", errors.New("SQL statement is empty after removing comments and blank lines")
	}
	return result, nil
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
		// Use the parent context (ctx), not the cancelled pollCtx.
		cancelCtx, cancel := context.WithTimeout(ctx, cancelTimeout)
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

// fetchAllRows collects all result rows, fetching additional chunks if needed.
func fetchAllRows(ctx context.Context, api sql.StatementExecutionInterface, resp *sql.StatementResponse) ([][]string, error) {
	if resp.Result == nil {
		return nil, nil
	}

	rows := append([][]string{}, resp.Result.DataArray...)

	totalChunks := 0
	if resp.Manifest != nil {
		totalChunks = resp.Manifest.TotalChunkCount
	}

	for chunk := 1; chunk < totalChunks; chunk++ {
		log.Debugf(ctx, "Fetching result chunk %d/%d for statement %s", chunk+1, totalChunks, resp.StatementId)
		chunkResp, err := api.GetStatementResultChunkNByStatementIdAndChunkIndex(ctx, resp.StatementId, chunk)
		if err != nil {
			return nil, fmt.Errorf("fetch result chunk %d: %w", chunk, err)
		}
		rows = append(rows, chunkResp.DataArray...)
	}

	return rows, nil
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
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
