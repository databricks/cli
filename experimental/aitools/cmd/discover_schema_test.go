package aitools

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	dbsql "github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"valid", "main.public.orders", "`main`.`public`.`orders`", ""},
		{"underscores ok", "_a.b_c.d_e", "`_a`.`b_c`.`d_e`", ""},
		{"missing parts", "public.orders", "", "expected CATALOG.SCHEMA.TABLE"},
		{"too many parts", "a.b.c.d", "", "expected CATALOG.SCHEMA.TABLE"},
		{"injection in catalog", "a;DROP--.b.c", "", "invalid SQL identifier"},
		{"backtick in name", "a.b.c`d", "", "invalid SQL identifier"},
		{"empty part", "a..c", "", "invalid SQL identifier"},
		{"starts with digit", "1main.public.orders", "", "invalid SQL identifier"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := quoteTableName(tc.in)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseDescribeResultSkipsMetadataRows(t *testing.T) {
	resp := &dbsql.StatementResponse{
		Result: &dbsql.ResultData{DataArray: [][]string{
			{"id", "BIGINT", ""},
			{"name", "STRING", ""},
			{"# Partition Information", "", ""},
			{"region", "STRING", ""},
			{"", "STRING", ""},
		}},
	}

	cols, types := parseDescribeResult(resp)
	assert.Equal(t, []string{"id", "name", "region"}, cols)
	assert.Equal(t, []string{"BIGINT", "STRING", "STRING"}, types)
}

func TestSQLGateRunPinsOnWaitTimeoutAndRecordsID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT 1" &&
			req.WaitTimeout == "0s" &&
			req.OnWaitTimeout == dbsql.ExecuteStatementRequestOnWaitTimeoutContinue
	})).Return(&dbsql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStateSucceeded},
		Result:      &dbsql.ResultData{DataArray: [][]string{{"1"}}},
	}, nil).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(2)

	resp, err := gate.run(ctx, w, "wh-1", "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", resp.StatementId)
	assert.Equal(t, []string{"stmt-1"}, gate.trackedIDs())
}

func TestSQLGateRunPropagatesFailedState(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&dbsql.StatementResponse{
		StatementId: "stmt-1",
		Status: &dbsql.StatementStatus{
			State: dbsql.StatementStateFailed,
			Error: &dbsql.ServiceError{ErrorCode: "SYNTAX_ERROR", Message: "near 'oops'"},
		},
	}, nil).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(2)

	_, err := gate.run(ctx, w, "wh-1", "SELECT oops")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SYNTAX_ERROR")
	// Even on failure, the id is recorded so a cancellation sweep can clean up.
	assert.Equal(t, []string{"stmt-1"}, gate.trackedIDs())
}

func TestSQLGateRunWrapsTransportError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).
		Return(nil, errors.New("network unreachable")).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(2)

	_, err := gate.run(ctx, w, "wh-1", "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute statement")
	assert.Contains(t, err.Error(), "network unreachable")
	assert.Empty(t, gate.trackedIDs(), "no id should be recorded when ExecuteStatement fails")
}

func TestSQLGateRunRespectsCancelledContext(t *testing.T) {
	// With ctx already cancelled, gate.run must not call any API method:
	// it bails at the semaphore-acquire select.
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	cancel()

	mockAPI := mocksql.NewMockStatementExecutionInterface(t)
	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(2)

	_, err := gate.run(ctx, w, "wh-1", "SELECT 1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDiscoverTableRunsSampleAndNullsConcurrently(t *testing.T) {
	// Deterministic barrier: both probes must enter before either is allowed
	// to leave. If gate.run/discoverTable serialized them, the first probe
	// would time out and return an error, which would surface as
	// "SAMPLE DATA: Error - " or "NULL COUNTS: Error - " in the output.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(req.Statement, "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: "stmt-desc",
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStateSucceeded},
		Result: &dbsql.ResultData{DataArray: [][]string{
			{"id", "BIGINT", ""},
			{"name", "STRING", ""},
		}},
	}, nil).Once()

	const numProbes = 2
	var dispatched atomic.Int32
	release := make(chan struct{})
	closeRelease := sync.OnceFunc(func() { close(release) })

	probe := func(ctx context.Context, req dbsql.ExecuteStatementRequest) (*dbsql.StatementResponse, error) {
		if dispatched.Add(1) == numProbes {
			closeRelease()
		}
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			return nil, errors.New("probe timeout: not running concurrently")
		}
		return &dbsql.StatementResponse{
			StatementId: "stmt-probe-" + req.Statement[:7],
			Status:      &dbsql.StatementStatus{State: dbsql.StatementStateSucceeded},
			Manifest:    &dbsql.ResultManifest{Schema: &dbsql.ResultSchema{Columns: []dbsql.ColumnInfo{{Name: "x"}}}},
			Result:      &dbsql.ResultData{DataArray: [][]string{{"0"}}},
		}, nil
	}

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(req.Statement, "SELECT *")
	})).RunAndReturn(probe).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(req.Statement, "SUM(CASE WHEN")
	})).RunAndReturn(probe).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(8)

	out, err := discoverTable(ctx, gate, w, "wh-1", "main.public.orders")
	require.NoError(t, err)
	assert.Equal(t, int32(numProbes), dispatched.Load(), "both probes should have entered concurrently")
	assert.NotContains(t, out, "Error - ", "no probe should have surfaced an error")
	assert.Contains(t, out, "COLUMNS:")
	assert.Contains(t, out, "SAMPLE DATA:")
	assert.Contains(t, out, "NULL COUNTS:")
}

func TestDiscoverTableSampleErrorDoesNotAbortNullCounts(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(req.Statement, "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: "stmt-desc",
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStateSucceeded},
		Result:      &dbsql.ResultData{DataArray: [][]string{{"id", "BIGINT", ""}}},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(req.Statement, "SELECT *")
	})).Return(&dbsql.StatementResponse{
		StatementId: "stmt-sample",
		Status: &dbsql.StatementStatus{
			State: dbsql.StatementStateFailed,
			Error: &dbsql.ServiceError{ErrorCode: "PERM", Message: "permission denied"},
		},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(req.Statement, "SUM(CASE WHEN")
	})).Return(&dbsql.StatementResponse{
		StatementId: "stmt-null",
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStateSucceeded},
		Manifest:    &dbsql.ResultManifest{Schema: &dbsql.ResultSchema{Columns: []dbsql.ColumnInfo{{Name: "total_rows"}, {Name: "id_nulls"}}}},
		Result:      &dbsql.ResultData{DataArray: [][]string{{"100", "0"}}},
	}, nil).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	gate := newSQLGate(8)

	out, err := discoverTable(ctx, gate, w, "wh-1", "main.public.orders")
	require.NoError(t, err)
	assert.Contains(t, out, "SAMPLE DATA: Error - ")
	assert.Contains(t, out, "permission denied")
	assert.Contains(t, out, "NULL COUNTS:")
	assert.Contains(t, out, "total_rows: 100")
}

func TestCancelDiscoverInFlightCallsAPIPerID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	for _, id := range []string{"stmt-a", "stmt-b", "stmt-c"} {
		mockAPI.EXPECT().CancelExecution(mock.Anything, dbsql.CancelExecutionRequest{
			StatementId: id,
		}).Return(nil).Once()
	}

	cancelDiscoverInFlight(ctx, mockAPI, []string{"stmt-a", "stmt-b", "stmt-c"})
}

func TestDiscoverSchemaConcurrencyRejection(t *testing.T) {
	for _, value := range []string{"0", "-1"} {
		t.Run(value, func(t *testing.T) {
			cmd := newDiscoverSchemaCmd()
			cmd.SetArgs([]string{"--concurrency", value, "main.public.orders"})
			err := cmd.Execute()
			require.ErrorIs(t, err, errInvalidBatchConcurrency)
		})
	}
}

func TestDiscoverSchemaInvalidTableNameRejectedBeforeWorkspaceClient(t *testing.T) {
	// PreRunE rejects malformed identifiers before MustWorkspaceClient runs,
	// so the test passes without any workspace mocking.
	cmd := newDiscoverSchemaCmd()
	cmd.SetArgs([]string{"not-three-parts"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected CATALOG.SCHEMA.TABLE")
}
