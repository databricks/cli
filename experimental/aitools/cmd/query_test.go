package aitools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/libs/sqlcli"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCleanSQL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{"plain", "SELECT 1", "SELECT 1"},
		{"double quoted", `"SELECT 1"`, "SELECT 1"},
		{"single quoted", `'SELECT 1'`, "SELECT 1"},
		{"strips comments", "-- comment\nSELECT 1", "SELECT 1"},
		{"strips empty lines", "\n\nSELECT 1\n\n", "SELECT 1"},
		{"multiline", "SELECT\n  1\nFROM\n  dual", "SELECT\n1\nFROM\ndual"},
		{"empty", "", ""},
		{"only comments", "-- comment\n-- another", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.out, cleanSQL(tc.in))
		})
	}
}

func TestExecuteAndPollImmediateSuccess(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.WarehouseId == "wh-123" && req.Statement == "SELECT 1" &&
			req.WaitTimeout == "0s" &&
			req.OnWaitTimeout == sql.ExecuteStatementRequestOnWaitTimeoutContinue
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "1"}}}},
		Result:      &sql.ResultData{DataArray: [][]string{{"1"}}},
	}, nil)

	resp, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	assert.Equal(t, "stmt-1", resp.StatementId)
}

func TestExecuteAndPollImmediateFailure(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{
				ErrorCode: "SYNTAX_ERROR",
				Message:   "near 'SELCT': syntax error",
			},
		},
	}, nil)

	_, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELCT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SYNTAX_ERROR")
	assert.Contains(t, err.Error(), "syntax error")
}

func TestExecuteAndPollWithPolling(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil)

	// First poll: still RUNNING.
	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateRunning},
	}, nil).Once()

	// Second poll: SUCCEEDED.
	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "result"}}}},
		Result:      &sql.ResultData{DataArray: [][]string{{"42"}}},
	}, nil).Once()

	resp, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELECT 42")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	assert.Equal(t, [][]string{{"42"}}, resp.Result.DataArray)
}

func TestExecuteAndPollFailsDuringPolling(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{ErrorCode: "RESOURCE_EXHAUSTED", Message: "warehouse unavailable"},
		},
	}, nil).Once()

	_, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESOURCE_EXHAUSTED")
}

func TestExecuteAndPollCancelledContextCallsCancelExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil)

	// CancelExecution must be called when context is cancelled (not just on
	// signal). Assert the RPC's own ctx is NOT cancelled, otherwise the SDK
	// would short-circuit on ctx.Err() and never reach the warehouse.
	mockAPI.EXPECT().CancelExecution(mock.MatchedBy(func(c context.Context) bool {
		return c.Err() == nil
	}), sql.CancelExecutionRequest{
		StatementId: "stmt-1",
	}).Return(nil).Once()

	cancel()

	_, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELECT 1")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
}

func TestPollStatementImmediateTerminal(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	resp := &sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "1"}}}},
		Result:      &sql.ResultData{DataArray: [][]string{{"1"}}},
	}

	pollResp, err := pollStatement(ctx, mockAPI, resp)
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateSucceeded, pollResp.Status.State)
	assert.Equal(t, "stmt-1", pollResp.StatementId)
}

func TestPollStatementTerminalFailureNotErrored(t *testing.T) {
	// pollStatement returns the response without erroring on failed terminal
	// states; callers (e.g. executeAndPoll) decide what to do via checkFailedState.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	resp := &sql.StatementResponse{
		StatementId: "stmt-1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{ErrorCode: "ERR", Message: "boom"},
		},
	}

	pollResp, err := pollStatement(ctx, mockAPI, resp)
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateFailed, pollResp.Status.State)
}

func TestPollStatementEventualSuccess(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	initial := &sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateRunning},
	}, nil).Once()

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Result:      &sql.ResultData{DataArray: [][]string{{"42"}}},
	}, nil).Once()

	pollResp, err := pollStatement(ctx, mockAPI, initial)
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateSucceeded, pollResp.Status.State)
	assert.Equal(t, [][]string{{"42"}}, pollResp.Result.DataArray)
}

func TestPollStatementContextCancellationDoesNotCancelServerSide(t *testing.T) {
	// The mock asserts (via t.Cleanup) that no unexpected calls are made.
	// Specifically, pollStatement must NOT call CancelExecution on context
	// cancellation; that is the caller's responsibility.
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	initial := &sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}

	cancel()

	pollResp, err := pollStatement(ctx, mockAPI, initial)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, pollResp)
}

func TestPollStatementGetErrorPropagated(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	initial := &sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").
		Return(nil, errors.New("network unreachable")).Once()

	pollResp, err := pollStatement(ctx, mockAPI, initial)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "poll statement status")
	assert.Contains(t, err.Error(), "network unreachable")
	assert.Nil(t, pollResp)
}

func TestResolveWarehouseIDWithFlag(t *testing.T) {
	ctx := t.Context()
	id, err := resolveWarehouseID(ctx, nil, "explicit-id")
	require.NoError(t, err)
	assert.Equal(t, "explicit-id", id)
}

func TestSelectQueryOutputMode(t *testing.T) {
	tests := []struct {
		name              string
		format            sqlcli.Format
		stdoutInteractive bool
		promptSupported   bool
		rowCount          int
		want              queryOutputMode
	}{
		{
			name:              "json flag always returns json",
			format:            sqlcli.OutputJSON,
			stdoutInteractive: true,
			promptSupported:   true,
			rowCount:          999,
			want:              queryOutputModeJSON,
		},
		{
			name:              "non interactive stdout returns json",
			format:            sqlcli.OutputText,
			stdoutInteractive: false,
			promptSupported:   true,
			rowCount:          5,
			want:              queryOutputModeJSON,
		},
		{
			name:              "missing stdin interactivity falls back to static table",
			format:            sqlcli.OutputText,
			stdoutInteractive: true,
			promptSupported:   false,
			rowCount:          sqlcli.StaticTableThreshold + 10,
			want:              queryOutputModeStaticTable,
		},
		{
			name:              "small results use static table",
			format:            sqlcli.OutputText,
			stdoutInteractive: true,
			promptSupported:   true,
			rowCount:          sqlcli.StaticTableThreshold,
			want:              queryOutputModeStaticTable,
		},
		{
			name:              "large results use interactive table",
			format:            sqlcli.OutputText,
			stdoutInteractive: true,
			promptSupported:   true,
			rowCount:          sqlcli.StaticTableThreshold + 1,
			want:              queryOutputModeInteractiveTable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := selectQueryOutputMode(tc.format, tc.stdoutInteractive, tc.promptSupported, tc.rowCount)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFetchAllRowsSingleChunk(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	resp := &sql.StatementResponse{
		StatementId: "stmt-1",
		Manifest:    &sql.ResultManifest{TotalChunkCount: 1},
		Result:      &sql.ResultData{DataArray: [][]string{{"1", "alice"}, {"2", "bob"}}},
	}

	rows, err := fetchAllRows(ctx, mockAPI, resp)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"1", "alice"}, {"2", "bob"}}, rows)
}

func TestFetchAllRowsMultiChunk(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	resp := &sql.StatementResponse{
		StatementId: "stmt-1",
		Manifest:    &sql.ResultManifest{TotalChunkCount: 3},
		Result:      &sql.ResultData{DataArray: [][]string{{"1", "a"}}},
	}

	mockAPI.EXPECT().GetStatementResultChunkNByStatementIdAndChunkIndex(mock.Anything, "stmt-1", 1).
		Return(&sql.ResultData{DataArray: [][]string{{"2", "b"}}}, nil).Once()
	mockAPI.EXPECT().GetStatementResultChunkNByStatementIdAndChunkIndex(mock.Anything, "stmt-1", 2).
		Return(&sql.ResultData{DataArray: [][]string{{"3", "c"}}}, nil).Once()

	rows, err := fetchAllRows(ctx, mockAPI, resp)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"1", "a"}, {"2", "b"}, {"3", "c"}}, rows)
}

func TestFetchAllRowsNilResult(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	resp := &sql.StatementResponse{StatementId: "stmt-1"}

	rows, err := fetchAllRows(ctx, mockAPI, resp)
	require.NoError(t, err)
	assert.Nil(t, rows)
}

func TestIsTerminalState(t *testing.T) {
	tests := []struct {
		state    sql.StatementState
		terminal bool
	}{
		{sql.StatementStateSucceeded, true},
		{sql.StatementStateFailed, true},
		{sql.StatementStateCanceled, true},
		{sql.StatementStateClosed, true},
		{sql.StatementStatePending, false},
		{sql.StatementStateRunning, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.state), func(t *testing.T) {
			status := &sql.StatementStatus{State: tc.state}
			assert.Equal(t, tc.terminal, isTerminalState(status))
		})
	}

	assert.False(t, isTerminalState(nil))
}

func TestCheckFailedState(t *testing.T) {
	assert.NoError(t, checkFailedState(nil))
	assert.NoError(t, checkFailedState(&sql.StatementStatus{State: sql.StatementStateSucceeded}))

	err := checkFailedState(&sql.StatementStatus{
		State: sql.StatementStateFailed,
		Error: &sql.ServiceError{ErrorCode: "ERR", Message: "bad"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ERR")
	assert.Contains(t, err.Error(), "bad")

	err = checkFailedState(&sql.StatementStatus{State: sql.StatementStateCanceled})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")

	err = checkFailedState(&sql.StatementStatus{State: sql.StatementStateClosed})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestCheckFailedStateMapKeyHint(t *testing.T) {
	err := checkFailedState(&sql.StatementStatus{
		State: sql.StatementStateFailed,
		Error: &sql.ServiceError{
			ErrorCode: "BAD_REQUEST",
			Message:   "[UNRESOLVED_MAP_KEY.WITH_SUGGESTION] Cannot resolve column",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Hint:")
	assert.Contains(t, err.Error(), "single quotes")
	assert.Contains(t, err.Error(), "--file")
}

func TestPollingConstants(t *testing.T) {
	assert.Equal(t, 1*time.Second, pollIntervalInitial)
	assert.Equal(t, 5*time.Second, pollIntervalMax)
	assert.Equal(t, 10*time.Second, cancelTimeout)
}

// newTestCmd creates a minimal cobra.Command for testing resolveSQLs.
func newTestCmd() *cobra.Command {
	return &cobra.Command{Use: "test"}
}

func TestResolveSQLsFromFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query.sql")
	err := os.WriteFile(path, []byte("SELECT 1"), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{path})
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 1"}, result)
}

func TestResolveSQLsFromFileFlagWithComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query.sql")
	err := os.WriteFile(path, []byte("-- header comment\nSELECT 1\n-- trailing"), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{path})
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 1"}, result)
}

func TestResolveSQLsMixedFileAndPositional(t *testing.T) {
	// --file paths are emitted before positional args, in flag order.
	dir := t.TempDir()
	path := filepath.Join(dir, "from-file.sql")
	err := os.WriteFile(path, []byte("SELECT 'from file'"), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{"SELECT 'from arg'"}, []string{path})
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 'from file'", "SELECT 'from arg'"}, result)
}

func TestResolveSQLsMultiplePositional(t *testing.T) {
	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{"SELECT 1", "SELECT 2", "SELECT 3"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 1", "SELECT 2", "SELECT 3"}, result)
}

func TestResolveSQLsMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.sql")
	pathB := filepath.Join(dir, "b.sql")
	require.NoError(t, os.WriteFile(pathA, []byte("SELECT 'a'"), 0o644))
	require.NoError(t, os.WriteFile(pathB, []byte("SELECT 'b'"), 0o644))

	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{pathA, pathB})
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 'a'", "SELECT 'b'"}, result)
}

func TestResolveSQLsFromPositionalArg(t *testing.T) {
	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{"SELECT 42"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 42"}, result)
}

func TestResolveSQLsAutoDetectsSQLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.sql")
	err := os.WriteFile(path, []byte("SELECT * FROM sales"), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{path}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT * FROM sales"}, result)
}

func TestResolveSQLsNonexistentSQLFileTreatedAsString(t *testing.T) {
	cmd := newTestCmd()
	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{"nonexistent.sql"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"nonexistent.sql"}, result)
}

func TestResolveSQLsUnreadableSQLFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.sql")
	err := os.WriteFile(path, []byte("SELECT 1"), 0o644)
	require.NoError(t, err)

	// Remove read permission to simulate an unreadable file.
	err = os.Chmod(path, 0o000)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	cmd := newTestCmd()
	_, err = resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{path}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read SQL file")
}

func TestResolveSQLsFromStdin(t *testing.T) {
	cmd := newTestCmd()
	cmd.SetIn(strings.NewReader("SELECT 1 FROM stdin_test"))

	result, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"SELECT 1 FROM stdin_test"}, result)
}

func TestResolveSQLsEmptyFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.sql")
	err := os.WriteFile(path, []byte(""), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	_, err = resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{path})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestResolveSQLsCommentsOnlyFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.sql")
	err := os.WriteFile(path, []byte("-- just a comment\n-- another"), 0o644)
	require.NoError(t, err)

	cmd := newTestCmd()
	_, err = resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{path})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestResolveSQLsBatchEmptyAtIndexReturnsIndexedError(t *testing.T) {
	cmd := newTestCmd()
	_, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, []string{"SELECT 1", "-- comment only", "SELECT 3"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SQL statement #2 is empty")
}

func TestResolveSQLsMissingFileReturnsError(t *testing.T) {
	cmd := newTestCmd()
	_, err := resolveSQLs(cmdio.MockDiscard(t.Context()), cmd, nil, []string{"/nonexistent/path/query.sql"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read SQL file")
}

func TestQueryCommandUnsupportedOutputReturnsError(t *testing.T) {
	cmd := newQueryCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"--output", "xml", "SELECT 1"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}

func TestQueryCommandBatchOutputRejection(t *testing.T) {
	// Multi-query mode is JSON-only. text and csv are rejected with an
	// actionable error before any API call.
	for _, format := range []string{"text", "csv"} {
		t.Run(format, func(t *testing.T) {
			cmd := newQueryCmd()
			cmd.PreRunE = nil
			cmd.SetArgs([]string{"--output", format, "SELECT 1", "SELECT 2"})
			err := cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "multiple queries require --output json")
		})
	}
}

func TestQueryCommandConcurrencyRejection(t *testing.T) {
	// errgroup.SetLimit(0) deadlocks; negative removes the cap entirely.
	// Both surprise users, so PreRunE rejects anything <= 0.
	for _, value := range []string{"0", "-1"} {
		t.Run(value, func(t *testing.T) {
			cmd := newQueryCmd()
			cmd.SetArgs([]string{"--concurrency", value, "--output", "json", "SELECT 1", "SELECT 2"})
			err := cmd.Execute()
			require.ErrorIs(t, err, errInvalidBatchConcurrency)
		})
	}
}

func TestQueryCommandOutputFlagIsCaseInsensitive(t *testing.T) {
	cmd := newQueryCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"--output", "JSON", "SELECT 1"})
	// "JSON" is lowercased and passes validation. The command proceeds to
	// WorkspaceClient and panics (no client in test), confirming validation passed.
	assert.Panics(t, func() { _ = cmd.Execute() })
}

func TestQueryCommandEnvVarOverridesDefault(t *testing.T) {
	cmd := newQueryCmd()
	cmd.PreRunE = nil
	ctx := env.Set(t.Context(), "DATABRICKS_OUTPUT_FORMAT", "json")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"SELECT 1"})
	// Env var "json" is valid, so validation passes and the command proceeds
	// to WorkspaceClient (panics because no client in test context).
	assert.Panics(t, func() { _ = cmd.Execute() })
}

func TestQueryCommandInvalidEnvVarIsIgnored(t *testing.T) {
	cmd := newQueryCmd()
	cmd.PreRunE = nil
	ctx := env.Set(t.Context(), "DATABRICKS_OUTPUT_FORMAT", "xml")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"SELECT 1"})
	// Invalid env value is silently ignored (falls back to default "text"),
	// so validation passes and the command proceeds to WorkspaceClient.
	assert.Panics(t, func() { _ = cmd.Execute() })
}

func TestQueryCommandExplicitFlagOverridesEnvVar(t *testing.T) {
	cmd := newQueryCmd()
	cmd.PreRunE = nil
	ctx := env.Set(t.Context(), "DATABRICKS_OUTPUT_FORMAT", "json")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--output", "csv", "SELECT 1"})
	// Explicit --output csv overrides env var. Validation passes,
	// command proceeds to WorkspaceClient.
	assert.Panics(t, func() { _ = cmd.Execute() })
}

func TestRenderCSVOutput(t *testing.T) {
	var buf strings.Builder
	err := renderCSV(&buf, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}})
	require.NoError(t, err)
	assert.Equal(t, "id,name\r\n1,alice\r\n2,bob\r\n", buf.String())
}

func TestRenderCSVHeadersOnlyWhenNoRows(t *testing.T) {
	var buf strings.Builder
	err := renderCSV(&buf, []string{"id", "name"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "id,name\r\n", buf.String())
}
