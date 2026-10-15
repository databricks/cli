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
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/utils"
	dbsql "github.com/databricks/sdk-go/statementexecution/v1"
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
	result := &sqlexec.Result{
		Rows: [][]string{
			{"id", "BIGINT", ""},
			{"name", "STRING", ""},
			{"# Partition Information", "", ""},
			{"region", "STRING", ""},
			{"", "STRING", ""},
		},
	}

	cols, types := parseDescribeResult(result)
	assert.Equal(t, []string{"id", "name", "region"}, cols)
	assert.Equal(t, []string{"BIGINT", "STRING", "STRING"}, types)
}

func TestSQLGateRunPinsOnWaitTimeoutAndRecordsID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return utils.Deref(req.Statement) == "SELECT 1" &&
			utils.Deref(req.WaitTimeout) == "0s" &&
			req.OnWaitTimeout == dbsql.TimeoutAction_Continue
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"1"}})},
	}, nil).Once()

	w := mockAPI
	gate := newSQLGate(2)

	result, err := gate.run(ctx, w, "wh-1", "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"1"}}, result.Rows)
	assert.Equal(t, []string{"stmt-1"}, gate.trackedIDs())
}

func TestSQLGateRunPropagatesFailedState(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-1"),
		Status: &dbsql.StatementStatus{
			State: dbsql.StatementStatus_State_Failed,
			Error: &dbsql.ServiceError{ErrorCode: "SYNTAX_ERROR", Message: new("near 'oops'")},
		},
	}, nil).Once()

	w := mockAPI
	gate := newSQLGate(2)

	_, err := gate.run(ctx, w, "wh-1", "SELECT oops")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SYNTAX_ERROR")
	// Even on failure, the id is recorded so a cancellation sweep can clean up.
	assert.Equal(t, []string{"stmt-1"}, gate.trackedIDs())
}

func TestSQLGateRunWrapsTransportError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).
		Return(nil, errors.New("network unreachable")).Once()

	w := mockAPI
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

	mockAPI := newMockStatementAPI(t)
	w := mockAPI
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
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-desc"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Result: &dbsql.ResultData{DataArray: rows([][]string{
			{"id", "BIGINT", ""},
			{"name", "STRING", ""},
		})},
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
			StatementId: new("stmt-probe-" + (*req.Statement)[:7]),
			Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
			Manifest:    &dbsql.ResultManifest{Schema: &dbsql.Schema{Columns: []dbsql.ColumnInfo{{Name: new("x")}}}},
			Result:      &dbsql.ResultData{DataArray: rows([][]string{{"0"}})},
		}, nil
	}

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "SELECT *")
	})).RunAndReturn(probe).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(utils.Deref(req.Statement), "SUM(CASE WHEN")
	})).RunAndReturn(probe).Once()

	w := mockAPI
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
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-desc"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"id", "BIGINT", ""}})},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "SELECT *")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-sample"),
		Status: &dbsql.StatementStatus{
			State: dbsql.StatementStatus_State_Failed,
			Error: &dbsql.ServiceError{ErrorCode: "PERM", Message: new("permission denied")},
		},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(utils.Deref(req.Statement), "SUM(CASE WHEN")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-null"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Manifest:    &dbsql.ResultManifest{Schema: &dbsql.Schema{Columns: []dbsql.ColumnInfo{{Name: new("total_rows")}, {Name: new("id_nulls")}}}},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"100", "0"}})},
	}, nil).Once()

	w := mockAPI
	gate := newSQLGate(8)

	out, err := discoverTable(ctx, gate, w, "wh-1", "main.public.orders")
	require.NoError(t, err)
	assert.Contains(t, out, "SAMPLE DATA: Error - ")
	assert.Contains(t, out, "permission denied")
	assert.Contains(t, out, "NULL COUNTS:")
	assert.Contains(t, out, "total_rows: 100")
}

func TestDiscoverTableEscapesBackticksInColumnNames(t *testing.T) {
	// Databricks/Delta DDL allows backticks in column names via doubled-
	// backtick escaping (e.g. CREATE TABLE t (`weird``col` STRING)). Without
	// escaping in the null-counts SQL the embedded backtick would terminate
	// the quoted identifier mid-string and produce a PARSE_SYNTAX_ERROR.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-desc"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Result: &dbsql.ResultData{DataArray: rows([][]string{
			{"weird`col", "STRING", ""},
		})},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "SELECT *")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-sample"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
	}, nil).Once()

	// Null-counts SQL must escape the embedded backtick. Both the identifier
	// and the alias positions must use the doubled form.
	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(utils.Deref(req.Statement), "`weird``col`") &&
			strings.Contains(utils.Deref(req.Statement), "`weird``col_nulls`") &&
			!strings.Contains(utils.Deref(req.Statement), "`weird`col`")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-null"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Manifest:    &dbsql.ResultManifest{Schema: &dbsql.Schema{Columns: []dbsql.ColumnInfo{{Name: new("total_rows")}, {Name: new("weird`col_nulls")}}}},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"5", "0"}})},
	}, nil).Once()

	w := mockAPI
	gate := newSQLGate(8)

	out, err := discoverTable(ctx, gate, w, "wh-1", "main.public.orders")
	require.NoError(t, err)
	assert.Contains(t, out, "weird`col")
	assert.NotContains(t, out, "Error - ")
}

func TestCancelDiscoverInFlightCallsAPIPerID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	for _, id := range []string{"stmt-a", "stmt-b", "stmt-c"} {
		mockAPI.EXPECT().CancelStatement(mock.Anything, id).Return(nil).Once()
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

func TestRunDiscoverSchemasFlagsTableFailureForExitCode(t *testing.T) {
	// runDiscoverSchemas must report any per-table failure via the bool
	// return so the caller can produce a non-zero exit. Without this signal
	// scripts and CI parse stdout to detect failure, which is brittle.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-bad"),
		Status: &dbsql.StatementStatus{
			State: dbsql.StatementStatus_State_Failed,
			Error: &dbsql.ServiceError{ErrorCode: "TABLE_OR_VIEW_NOT_FOUND", Message: new("no such table")},
		},
	}, nil).Once()

	w := mockAPI
	output, anyFailed := runDiscoverSchemas(ctx, newSQLGate(8), w, "wh-1", []string{"main.public.missing"})

	assert.True(t, anyFailed)
	assert.Contains(t, output, "Error discovering main.public.missing")
	assert.Contains(t, output, "TABLE_OR_VIEW_NOT_FOUND")
}

func TestRunDiscoverSchemasAllSucceedReturnsFalse(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "DESCRIBE TABLE")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-desc"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"id", "BIGINT", ""}})},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.HasPrefix(utils.Deref(req.Statement), "SELECT *")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-sample"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req dbsql.ExecuteStatementRequest) bool {
		return strings.Contains(utils.Deref(req.Statement), "SUM(CASE WHEN")
	})).Return(&dbsql.StatementResponse{
		StatementId: new("stmt-null"),
		Status:      &dbsql.StatementStatus{State: dbsql.StatementStatus_State_Succeeded},
		Manifest:    &dbsql.ResultManifest{Schema: &dbsql.Schema{Columns: []dbsql.ColumnInfo{{Name: new("total_rows")}, {Name: new("id_nulls")}}}},
		Result:      &dbsql.ResultData{DataArray: rows([][]string{{"7", "0"}})},
	}, nil).Once()

	w := mockAPI
	output, anyFailed := runDiscoverSchemas(ctx, newSQLGate(8), w, "wh-1", []string{"main.public.orders"})

	assert.False(t, anyFailed)
	assert.Contains(t, output, "COLUMNS:")
	assert.NotContains(t, output, "Error discovering")
}
