package aitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"golang.org/x/sync/errgroup"
)

// defaultBatchConcurrency caps in-flight statements when --concurrency is unset.
// Matches the default used by cmd/fs/cp.go for similar fan-out work.
const defaultBatchConcurrency = 8

// errInvalidBatchConcurrency is returned when --concurrency is set to a value
// that errgroup.SetLimit can't honor (0 deadlocks, negative removes the cap).
var errInvalidBatchConcurrency = errors.New("--concurrency must be at least 1")

// batchResult is the per-statement payload emitted in batch mode JSON output.
// State is the server-reported terminal state. Error is set whenever the
// statement did not produce usable rows, regardless of state, so consumers
// can branch on `error == null` alone.
type batchResult struct {
	SQL         string             `json:"sql"`
	StatementID string             `json:"statement_id,omitempty"`
	State       sql.StatementState `json:"state,omitempty"`
	ElapsedMs   int64              `json:"elapsed_ms"`
	Columns     []string           `json:"columns,omitempty"`
	Rows        [][]string         `json:"rows,omitempty"`
	Error       *batchResultError  `json:"error,omitempty"`
}

// batchResultError captures user-visible error info for a failed statement.
type batchResultError struct {
	Message   string `json:"message"`
	ErrorCode string `json:"error_code,omitempty"`
}

// executeBatch submits sqls against the warehouse in parallel, polls each to
// completion, and returns one batchResult per input in input order.
//
// Individual statement failures do not abort siblings; failures are encoded in
// the per-result Error field so callers can render partial results.
//
// On context cancellation (Ctrl+C or parent context), still-running statements
// are cancelled server-side via CancelExecution. Statements that finished
// before cancellation are left as-is.
func executeBatch(ctx context.Context, api sql.StatementExecutionInterface, warehouseID string, sqls []string, concurrency int) []batchResult {
	pollCtx, pollCancel := context.WithCancel(ctx)
	defer pollCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			log.Infof(ctx, "Received interrupt, cancelling %d in-flight queries", len(sqls))
			pollCancel()
		case <-pollCtx.Done():
		}
	}()

	sp := cmdio.NewSpinner(pollCtx)
	defer sp.Close()
	sp.Update(fmt.Sprintf("Executing %d queries...", len(sqls)))

	var completed atomic.Int64
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				sp.Update(fmt.Sprintf("Executing %d queries... (%d/%d done)", len(sqls), completed.Load(), len(sqls)))
			}
		}
	}()

	results := make([]batchResult, len(sqls))
	// Each goroutine writes to a distinct slot, safe without a mutex.
	// We read after g.Wait(), establishing happens-before for all writes.
	statementIDs := make([]string, len(sqls))

	g := new(errgroup.Group)
	g.SetLimit(concurrency)
	for i, sqlStr := range sqls {
		g.Go(func() error {
			results[i] = runOneBatchQuery(pollCtx, api, warehouseID, sqlStr, statementIDs, i)
			completed.Add(1)
			return nil
		})
	}
	_ = g.Wait()

	// pollStatement is a pure helper that returns ctx.Err() on cancellation
	// without touching the server. Sweep any not-yet-terminal statements here.
	if pollCtx.Err() != nil {
		cancelInFlight(ctx, api, statementIDs, results)
	}

	return results
}

// runOneBatchQuery submits one SQL, polls to completion, and returns its
// batchResult. All errors are encoded into the result; never returns an error.
func runOneBatchQuery(ctx context.Context, api sql.StatementExecutionInterface, warehouseID, sqlStr string, statementIDs []string, idx int) batchResult {
	start := time.Now()
	result := batchResult{SQL: sqlStr}

	resp, err := api.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		WarehouseId:   warehouseID,
		Statement:     sqlStr,
		WaitTimeout:   "0s",
		OnWaitTimeout: sql.ExecuteStatementRequestOnWaitTimeoutContinue,
	})
	if err != nil {
		if ctx.Err() != nil {
			result.State = sql.StatementStateCanceled
			result.Error = &batchResultError{Message: "submission cancelled"}
		} else {
			result.State = sql.StatementStateFailed
			result.Error = &batchResultError{Message: fmt.Sprintf("execute statement: %v", err)}
		}
		result.ElapsedMs = time.Since(start).Milliseconds()
		return result
	}

	statementIDs[idx] = resp.StatementId
	result.StatementID = resp.StatementId

	pollResp, err := pollStatement(ctx, api, resp)
	if err != nil {
		if ctx.Err() != nil {
			result.State = sql.StatementStateCanceled
			result.Error = &batchResultError{Message: "cancelled"}
		} else {
			result.State = sql.StatementStateFailed
			result.Error = &batchResultError{Message: err.Error()}
		}
		result.ElapsedMs = time.Since(start).Milliseconds()
		return result
	}

	if pollResp.Status != nil {
		result.State = pollResp.Status.State
	}

	if result.State != sql.StatementStateSucceeded {
		result.Error = &batchResultError{}
		if pollResp.Status != nil && pollResp.Status.Error != nil {
			result.Error.Message = pollResp.Status.Error.Message
			result.Error.ErrorCode = string(pollResp.Status.Error.ErrorCode)
		} else {
			result.Error.Message = fmt.Sprintf("query reached terminal state %s", result.State)
		}
		result.ElapsedMs = time.Since(start).Milliseconds()
		return result
	}

	result.Columns = extractColumns(pollResp.Manifest)
	rows, err := fetchAllRows(ctx, api, pollResp)
	if err != nil {
		result.Error = &batchResultError{Message: fmt.Sprintf("fetch rows: %v", err)}
		result.ElapsedMs = time.Since(start).Milliseconds()
		return result
	}
	result.Rows = rows
	result.ElapsedMs = time.Since(start).Milliseconds()
	return result
}

// cancelInFlight sends CancelExecution for every statement that didn't reach
// a terminal state server-side before context cancellation. Best effort: errors
// are logged at warn but don't fail the batch.
func cancelInFlight(ctx context.Context, api sql.StatementExecutionInterface, statementIDs []string, results []batchResult) {
	var cancelled int
	for i, sid := range statementIDs {
		if sid == "" {
			continue
		}
		switch results[i].State {
		case sql.StatementStateSucceeded, sql.StatementStateFailed, sql.StatementStateClosed:
			continue
		case sql.StatementStateCanceled, sql.StatementStatePending, sql.StatementStateRunning:
			// Either still running server-side, or our internal "canceled"
			// marker meaning the goroutine bailed without telling the server.
			// Either way, send CancelExecution.
		}
		// Detach from the inbound ctx (which is typically already cancelled by
		// the time we reach this sweep): WithoutCancel keeps the caller's
		// values but drops the cancellation signal so the cancel RPC actually
		// reaches the warehouse instead of short-circuiting on ctx.Err().
		cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cancelTimeout)
		if err := api.CancelExecution(cancelCtx, sql.CancelExecutionRequest{StatementId: sid}); err != nil {
			log.Warnf(ctx, "Failed to cancel statement %s: %v", sid, err)
		}
		cancel()
		cancelled++
	}
	if cancelled > 0 {
		cmdio.LogString(ctx, fmt.Sprintf("Cancelled %d in-flight queries.", cancelled))
	}
}
