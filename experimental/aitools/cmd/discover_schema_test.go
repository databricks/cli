package aitools

import (
	"context"
	"errors"
	"strings"
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
			{"# Partition Information", "", ""}, // metadata divider, skip
			{"region", "STRING", ""},
			{"", "STRING", ""}, // empty col name, skip
		}},
	}

	cols, types := parseDescribeResult(resp)
	assert.Equal(t, []string{"id", "name", "region"}, cols)
	assert.Equal(t, []string{"BIGINT", "STRING", "STRING"}, types)
}

func TestExecuteSQLUsesPollStatementAndPinsOnWaitTimeout(t *testing.T) {
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
	resp, err := executeSQL(ctx, w, "wh-1", "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", resp.StatementId)
}

func TestExecuteSQLPropagatesFailedState(t *testing.T) {
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
	_, err := executeSQL(ctx, w, "wh-1", "SELECT oops")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SYNTAX_ERROR")
}

func TestExecuteSQLWrapsTransportError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).
		Return(nil, errors.New("network unreachable")).Once()

	w := &databricks.WorkspaceClient{StatementExecution: mockAPI}
	_, err := executeSQL(ctx, w, "wh-1", "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute statement")
	assert.Contains(t, err.Error(), "network unreachable")
}

func TestDiscoverTableRunsSampleAndNullsInParallel(t *testing.T) {
	// After DESCRIBE returns, sample SELECT and null counts must run in
	// parallel, not back-to-back. Each mocked probe blocks briefly so an
	// atomic counter can observe peak in-flight calls.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	var inFlight, peak atomic.Int32

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

	probe := func(ctx context.Context, req dbsql.ExecuteStatementRequest) (*dbsql.StatementResponse, error) {
		n := inFlight.Add(1)
		for {
			cur := peak.Load()
			if n <= cur || peak.CompareAndSwap(cur, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		inFlight.Add(-1)
		return &dbsql.StatementResponse{
			StatementId: "stmt-probe",
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
	out, err := discoverTable(ctx, w, "wh-1", "main.public.orders")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, peak.Load(), int32(2), "sample and null-count probes should run concurrently")
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
	out, err := discoverTable(ctx, w, "wh-1", "main.public.orders")
	require.NoError(t, err)
	assert.Contains(t, out, "SAMPLE DATA: Error - ")
	assert.Contains(t, out, "permission denied")
	assert.Contains(t, out, "NULL COUNTS:")
	assert.Contains(t, out, "total_rows: 100")
}

func TestDiscoverSchemaConcurrencyZeroRejected(t *testing.T) {
	cmd := newDiscoverSchemaCmd()
	cmd.SetArgs([]string{"--concurrency", "0", "main.public.orders"})
	err := cmd.Execute()
	require.ErrorIs(t, err, errInvalidBatchConcurrency)
}

func TestDiscoverSchemaConcurrencyNegativeRejected(t *testing.T) {
	cmd := newDiscoverSchemaCmd()
	cmd.SetArgs([]string{"--concurrency", "-1", "main.public.orders"})
	err := cmd.Execute()
	require.ErrorIs(t, err, errInvalidBatchConcurrency)
}

func TestDiscoverSchemaInvalidTableNameRejected(t *testing.T) {
	cmd := newDiscoverSchemaCmd()
	cmd.PreRunE = nil // skip workspace client requirement
	cmd.SetArgs([]string{"not-three-parts"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected CATALOG.SCHEMA.TABLE")
}
