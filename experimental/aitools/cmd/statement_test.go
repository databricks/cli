package aitools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubmitStatementReturnsHandle(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.WarehouseId == "wh-1" && req.Statement == "SELECT 1" &&
			req.WaitTimeout == "0s" &&
			req.OnWaitTimeout == sql.ExecuteStatementRequestOnWaitTimeoutContinue
	})).Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil).Once()

	info, err := submitStatement(ctx, mockAPI, "SELECT 1", "wh-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStatePending, info.State)
	assert.Equal(t, "wh-1", info.WarehouseID)
}

func TestSubmitStatementWrapsTransportError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).
		Return(nil, errors.New("network unreachable")).Once()

	_, err := submitStatement(ctx, mockAPI, "SELECT 1", "wh-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute statement")
	assert.Contains(t, err.Error(), "network unreachable")
}

func TestGetStatementResultPolls(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateRunning},
	}, nil).Once()

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "n"}}}, TotalChunkCount: 1},
		Result:      &sql.ResultData{DataArray: [][]string{{"42"}}},
	}, nil).Once()

	info, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStateSucceeded, info.State)
	assert.Equal(t, []string{"n"}, info.Columns)
	assert.Equal(t, [][]string{{"42"}}, info.Rows)
	assert.Nil(t, info.Error)
}

func TestGetStatementResultFailedStateReportsError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{
				ErrorCode: "SYNTAX_ERROR",
				Message:   "near 'bad': syntax error",
			},
		},
	}, nil).Once()

	info, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateFailed, info.State)
	assert.Nil(t, info.Rows)
	require.NotNil(t, info.Error)
	assert.Equal(t, "SYNTAX_ERROR", info.Error.ErrorCode)
	assert.Contains(t, info.Error.Message, "syntax error")
}

func TestGetStatementResultDoesNotCancelServerSideOnContextCancel(t *testing.T) {
	// 'statement get' is a poll-only operation: ctx cancellation must NOT
	// trigger CancelExecution. The mock asserts (via t.Cleanup) that no
	// unexpected calls happen.
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStatePending},
	}, nil).Once()

	cancel()

	_, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetStatementStatusSinglePoll(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status:      &sql.StatementStatus{State: sql.StatementStateRunning},
	}, nil).Once()

	info, err := getStatementStatus(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStateRunning, info.State)
	assert.Nil(t, info.Error)
}

func TestGetStatementStatusReportsError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().GetStatementByStatementId(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: "stmt-1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{
				ErrorCode: "TIMEOUT",
				Message:   "warehouse timed out",
			},
		},
	}, nil).Once()

	info, err := getStatementStatus(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateFailed, info.State)
	require.NotNil(t, info.Error)
	assert.Equal(t, "TIMEOUT", info.Error.ErrorCode)
}

func TestCancelStatementExecutionCallsAPI(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().CancelExecution(mock.Anything, sql.CancelExecutionRequest{
		StatementId: "stmt-1",
	}).Return(nil).Once()

	info, err := cancelStatementExecution(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStateCanceled, info.State)
}

func TestCancelStatementExecutionWrapsAPIError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().CancelExecution(mock.Anything, mock.Anything).
		Return(errors.New("not found")).Once()

	_, err := cancelStatementExecution(ctx, mockAPI, "stmt-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel statement")
	assert.Contains(t, err.Error(), "not found")
}

func TestRenderStatementInfo(t *testing.T) {
	info := statementInfo{
		StatementID: "stmt-1",
		State:       sql.StatementStateSucceeded,
		WarehouseID: "wh-1",
		Columns:     []string{"n"},
		Rows:        [][]string{{"42"}},
	}

	var buf strings.Builder
	require.NoError(t, renderStatementInfo(&buf, info))

	output := buf.String()
	assert.Contains(t, output, `"statement_id": "stmt-1"`)
	assert.Contains(t, output, `"state": "SUCCEEDED"`)
	assert.Contains(t, output, `"warehouse_id": "wh-1"`)
	assert.Contains(t, output, `"columns": [`)
	assert.Contains(t, output, `"rows": [`)
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestRenderStatementInfoOmitsEmptyFields(t *testing.T) {
	// Cancel-style payload: only statement_id + state.
	info := statementInfo{
		StatementID: "stmt-1",
		State:       sql.StatementStateCanceled,
	}

	var buf strings.Builder
	require.NoError(t, renderStatementInfo(&buf, info))

	output := buf.String()
	assert.Contains(t, output, `"statement_id": "stmt-1"`)
	assert.Contains(t, output, `"state": "CANCELED"`)
	assert.NotContains(t, output, `"warehouse_id"`)
	assert.NotContains(t, output, `"columns"`)
	assert.NotContains(t, output, `"rows"`)
	assert.NotContains(t, output, `"error"`)
}

func TestStatementSubmitRejectsMultipleSQLs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")
	require.NoError(t, os.WriteFile(path, []byte("SELECT 1"), 0o644))

	cmd := newStatementSubmitCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"--file", path, "SELECT 2"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestStatementSubmitArgsBound(t *testing.T) {
	// MaximumNArgs(1) means cobra rejects 2+ positionals at parse time.
	cmd := newStatementSubmitCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"SELECT 1", "SELECT 2"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestStatementGetRequiresStatementID(t *testing.T) {
	cmd := newStatementGetCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestStatementCancelRequiresStatementID(t *testing.T) {
	cmd := newStatementCancelCmd()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
}
