package mcp

import (
	"context"
	"testing"
	"time"

	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/cli/libs/cmdio"
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
	ctx := cmdio.MockDiscard(context.Background())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.WarehouseId == "wh-123" && req.Statement == "SELECT 1" && req.WaitTimeout == "0s"
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
	ctx := cmdio.MockDiscard(context.Background())
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
	ctx := cmdio.MockDiscard(context.Background())
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
	ctx := cmdio.MockDiscard(context.Background())
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

func TestExecuteAndPollCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(context.Background()))
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil)

	cancel()

	_, err := executeAndPoll(ctx, mockAPI, "wh-123", "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestResolveWarehouseIDWithFlag(t *testing.T) {
	ctx := context.Background()
	id, err := resolveWarehouseID(ctx, nil, "explicit-id")
	require.NoError(t, err)
	assert.Equal(t, "explicit-id", id)
}

func TestFormatQueryResultNoResults(t *testing.T) {
	resp := &sql.StatementResponse{
		Status: &sql.StatementStatus{State: sql.StatementStateSucceeded},
	}
	output, err := formatQueryResult(resp)
	require.NoError(t, err)
	assert.Contains(t, output, "no results")
}

func TestFormatQueryResultWithData(t *testing.T) {
	resp := &sql.StatementResponse{
		Status: &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest: &sql.ResultManifest{
			Schema: &sql.ResultSchema{
				Columns: []sql.ColumnInfo{{Name: "id"}, {Name: "name"}},
			},
		},
		Result: &sql.ResultData{
			DataArray: [][]string{{"1", "alice"}, {"2", "bob"}},
		},
	}
	output, err := formatQueryResult(resp)
	require.NoError(t, err)
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "bob")
	assert.Contains(t, output, "Row count: 2")
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

func TestPollingConstants(t *testing.T) {
	assert.Equal(t, 1*time.Second, pollIntervalInitial)
	assert.Equal(t, 5*time.Second, pollIntervalMax)
	assert.Equal(t, 10*time.Second, cancelTimeout)
}
