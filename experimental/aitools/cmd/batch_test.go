package aitools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRenderBatchJSON(t *testing.T) {
	results := []batchResult{
		{
			SQL:         "SELECT 1",
			StatementID: "stmt-1",
			State:       sql.StatementStateSucceeded,
			ElapsedMs:   42,
			Columns:     []string{"n"},
			Rows:        [][]string{{"1"}},
		},
		{
			SQL:         "SELECT bad_syntax",
			StatementID: "stmt-2",
			State:       sql.StatementStateFailed,
			ElapsedMs:   12,
			Error: &batchResultError{
				Message:   "near 'bad_syntax': syntax error",
				ErrorCode: "SYNTAX_ERROR",
			},
		},
	}

	var buf strings.Builder
	err := renderBatchJSON(&buf, results)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"sql": "SELECT 1"`)
	assert.Contains(t, output, `"statement_id": "stmt-1"`)
	assert.Contains(t, output, `"state": "SUCCEEDED"`)
	assert.Contains(t, output, `"elapsed_ms": 42`)
	assert.Contains(t, output, `"columns": [`)
	assert.Contains(t, output, `"rows": [`)
	assert.Contains(t, output, `"sql": "SELECT bad_syntax"`)
	assert.Contains(t, output, `"error": {`)
	assert.Contains(t, output, `"error_code": "SYNTAX_ERROR"`)
	// Trailing newline.
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestExecuteBatchAllSucceed(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	sqls := []string{"SELECT 1", "SELECT 2", "SELECT 3"}
	for i, sqlStr := range sqls {
		sid := fmt.Sprintf("stmt-%d", i+1)
		mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
			return req.Statement == sqlStr
		})).Return(&sql.StatementResponse{
			StatementId: sid,
			Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
			Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "n"}}}},
			Result:      &sql.ResultData{DataArray: [][]string{{strconv.Itoa(i + 1)}}},
		}, nil).Once()
	}

	results := executeBatch(ctx, mockAPI, "wh-123", sqls, 8)

	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, sqls[i], r.SQL, "result %d sql", i)
		assert.Equal(t, sql.StatementStateSucceeded, r.State, "result %d state", i)
		assert.Nil(t, r.Error, "result %d error", i)
		assert.Equal(t, []string{"n"}, r.Columns, "result %d columns", i)
		assert.Equal(t, [][]string{{strconv.Itoa(i + 1)}}, r.Rows, "result %d rows", i)
		assert.NotEmpty(t, r.StatementID, "result %d statement_id", i)
	}
}

func TestExecuteBatchPartialFailure(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT 1"
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-good",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "n"}}}},
		Result:      &sql.ResultData{DataArray: [][]string{{"1"}}},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT bad"
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-bad",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{
				ErrorCode: "SYNTAX_ERROR",
				Message:   "near 'bad': syntax error",
			},
		},
	}, nil).Once()

	results := executeBatch(ctx, mockAPI, "wh-123", []string{"SELECT 1", "SELECT bad"}, 8)

	require.Len(t, results, 2)
	assert.Nil(t, results[0].Error)
	assert.Equal(t, sql.StatementStateSucceeded, results[0].State)

	require.NotNil(t, results[1].Error)
	assert.Equal(t, sql.StatementStateFailed, results[1].State)
	assert.Equal(t, "SYNTAX_ERROR", results[1].Error.ErrorCode)
	assert.Contains(t, results[1].Error.Message, "syntax error")
}

func TestExecuteBatchSubmissionFailure(t *testing.T) {
	// ExecuteStatement transport error is encoded into the per-result error,
	// not propagated up to abort siblings.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT good"
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-good",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
	}, nil).Once()

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT broken"
	})).Return(nil, errors.New("network unreachable")).Once()

	results := executeBatch(ctx, mockAPI, "wh-123", []string{"SELECT good", "SELECT broken"}, 8)

	require.Len(t, results, 2)
	assert.Nil(t, results[0].Error)
	require.NotNil(t, results[1].Error)
	assert.Contains(t, results[1].Error.Message, "execute statement")
	assert.Contains(t, results[1].Error.Message, "network unreachable")
	assert.Empty(t, results[1].StatementID)
}

func TestExecuteBatchSetsOnWaitTimeoutContinue(t *testing.T) {
	// Guards against a silent SDK default flip from CONTINUE to CANCEL.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.WaitTimeout == "0s" && req.OnWaitTimeout == sql.ExecuteStatementRequestOnWaitTimeoutContinue
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-x",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
	}, nil).Times(2)

	results := executeBatch(ctx, mockAPI, "wh-123", []string{"q1", "q2"}, 8)
	require.Len(t, results, 2)
}

func TestExecuteBatchPreservesInputOrder(t *testing.T) {
	// Index 0 is slow (PENDING then SUCCEEDED on first poll); 1 and 2 are
	// immediate. Despite the staggered completion, results stay in input order.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.Statement == "SELECT 'slow'"
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-slow",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil).Once()

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-slow").Return(&sql.StatementResponse{
		StatementId: "stmt-slow",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
	}, nil).Once()

	for i, sqlStr := range []string{"SELECT 'fast1'", "SELECT 'fast2'"} {
		sid := fmt.Sprintf("stmt-fast-%d", i+1)
		mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
			return req.Statement == sqlStr
		})).Return(&sql.StatementResponse{
			StatementId: sid,
			Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		}, nil).Once()
	}

	sqls := []string{"SELECT 'slow'", "SELECT 'fast1'", "SELECT 'fast2'"}
	results := executeBatch(ctx, mockAPI, "wh-1", sqls, 8)

	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, sqls[i], r.SQL, "result %d", i)
		assert.Equal(t, sql.StatementStateSucceeded, r.State, "result %d", i)
	}
}

func TestExecuteBatchContextCancellationCancelsInFlight(t *testing.T) {
	// All statements are PENDING when the context is cancelled. cancelInFlight
	// sweeps the in-flight set with CancelExecution.
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	for i, sqlStr := range []string{"q1", "q2", "q3"} {
		sid := fmt.Sprintf("stmt-%d", i+1)
		mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
			return req.Statement == sqlStr
		})).Return(&sql.StatementResponse{
			StatementId: sid,
			Status:      &sql.StatementStatus{State: sql.StatementStatePending},
		}, nil).Once()

		mockAPI.EXPECT().CancelExecution(mock.Anything, sql.CancelExecutionRequest{
			StatementId: sid,
		}).Return(nil).Once()
	}

	cancel()

	results := executeBatch(ctx, mockAPI, "wh", []string{"q1", "q2", "q3"}, 8)

	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, sql.StatementStateCanceled, r.State, "result %d state", i)
		require.NotNil(t, r.Error, "result %d error", i)
	}
}
