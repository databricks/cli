package aitools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/utils"
	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubmitStatementReturnsHandle(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return utils.Deref(req.WarehouseId) == "wh-1" && utils.Deref(req.Statement) == "SELECT 1" &&
			utils.Deref(req.WaitTimeout) == "0s" &&
			req.OnWaitTimeout == sql.TimeoutAction_Continue
	})).Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Pending},
	}, nil).Once()

	info, err := submitStatement(ctx, mockAPI, "SELECT 1", "wh-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStatus_State_Pending, info.State)
	assert.Equal(t, "wh-1", info.WarehouseID)
}

func TestSubmitStatementPassesParameters(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	params := []sql.StatementParameter{
		{Name: new("since"), Type: new("DATE"), Value: new("2026-01-01")},
	}

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return assert.ObjectsAreEqual(params, req.Parameters)
	})).Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Pending},
	}, nil).Once()

	info, err := submitStatement(ctx, mockAPI, "SELECT * FROM events WHERE ts > :since", "wh-1", params)
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
}

func TestSubmitStatementWrapsTransportError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).
		Return(nil, errors.New("network unreachable")).Once()

	_, err := submitStatement(ctx, mockAPI, "SELECT 1", "wh-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute statement")
	assert.Contains(t, err.Error(), "network unreachable")
}

func TestGetStatementResultPolls(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Running},
	}, nil).Once()

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Succeeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.Schema{Columns: []sql.ColumnInfo{{Name: new("n")}}}, TotalChunkCount: new(1)},
		Result:      &sql.ResultData{DataArray: rows([][]string{{"42"}})},
	}, nil).Once()

	info, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStatus_State_Succeeded, info.State)
	assert.Equal(t, []string{"n"}, info.Columns)
	assert.Equal(t, [][]string{{"42"}}, info.Rows)
	assert.Nil(t, info.Error)
}

func TestGetStatementResultFailedStateReportsError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status: &sql.StatementStatus{
			State: sql.StatementStatus_State_Failed,
			Error: &sql.ServiceError{
				ErrorCode: "SYNTAX_ERROR",
				Message:   new("near 'bad': syntax error"),
			},
		},
	}, nil).Once()

	info, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStatus_State_Failed, info.State)
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
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Pending},
	}, nil).Once()

	cancel()

	_, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetStatementResultChunkFetchFailureRendersPartialInfo(t *testing.T) {
	// SUCCEEDED state but a later chunk fetch fails (network blip, throttle,
	// 5xx). getStatementResult should surface this as a structured error on
	// the same statementInfo so the caller still gets parseable JSON with the
	// statement_id, instead of returning a raw Go error that RunE would
	// discard along with the populated info.
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Succeeded},
		Manifest: &sql.ResultManifest{
			Schema:          &sql.Schema{Columns: []sql.ColumnInfo{{Name: new("n")}}},
			TotalChunkCount: new(2),
		},
		Result: &sql.ResultData{DataArray: rows([][]string{{"1"}})},
	}, nil).Once()

	mockAPI.EXPECT().GetResultData(mock.Anything, "stmt-1", 1).
		Return(nil, errors.New("network blip")).Once()

	info, err := getStatementResult(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStatus_State_Succeeded, info.State)
	require.NotNil(t, info.Error)
	assert.Contains(t, info.Error.Message, "fetch result rows")
	assert.Contains(t, info.Error.Message, "network blip")
	// Column metadata is known from the manifest before the failed chunk fetch,
	// so it is still surfaced alongside the error.
	assert.Equal(t, []string{"n"}, info.Columns)
}

func TestGetStatementStatusSinglePoll(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Running},
	}, nil).Once()

	info, err := getStatementStatus(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStatus_State_Running, info.State)
	assert.Nil(t, info.Error)
}

func TestGetStatementStatusReportsError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().GetStatementResult(mock.Anything, "stmt-1").Return(&sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status: &sql.StatementStatus{
			State: sql.StatementStatus_State_Failed,
			Error: &sql.ServiceError{
				ErrorCode: "TIMEOUT",
				Message:   new("warehouse timed out"),
			},
		},
	}, nil).Once()

	info, err := getStatementStatus(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStatus_State_Failed, info.State)
	require.NotNil(t, info.Error)
	assert.Equal(t, "TIMEOUT", info.Error.ErrorCode)
}

func TestCancelStatementExecutionCallsAPI(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().CancelStatement(mock.Anything, "stmt-1").Return(nil).Once()

	info, err := cancelStatementExecution(ctx, mockAPI, "stmt-1")
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", info.StatementID)
	assert.Equal(t, sql.StatementStatus_State_Canceled, info.State)
}

func TestCancelStatementExecutionWrapsAPIError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := newMockStatementAPI(t)

	mockAPI.EXPECT().CancelStatement(mock.Anything, mock.Anything).
		Return(errors.New("not found")).Once()

	_, err := cancelStatementExecution(ctx, mockAPI, "stmt-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel statement")
	assert.Contains(t, err.Error(), "not found")
}

func TestRenderStatementInfo(t *testing.T) {
	tests := []struct {
		name        string
		info        statementInfo
		mustHave    []string
		mustNotHave []string
	}{
		{
			name: "full payload renders every populated field",
			info: statementInfo{
				StatementID: "stmt-1",
				State:       sql.StatementStatus_State_Succeeded,
				WarehouseID: "wh-1",
				Columns:     []string{"n"},
				Rows:        [][]string{{"42"}},
			},
			mustHave: []string{
				`"statement_id": "stmt-1"`,
				`"state": "SUCCEEDED"`,
				`"warehouse_id": "wh-1"`,
				`"columns": [`,
				`"rows": [`,
			},
		},
		{
			name: "cancel-style payload omits unset fields",
			info: statementInfo{
				StatementID: "stmt-1",
				State:       sql.StatementStatus_State_Canceled,
			},
			mustHave: []string{
				`"statement_id": "stmt-1"`,
				`"state": "CANCELED"`,
			},
			mustNotHave: []string{`"warehouse_id"`, `"columns"`, `"rows"`, `"error"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			require.NoError(t, renderStatementInfo(&buf, tc.info))
			out := buf.String()
			for _, want := range tc.mustHave {
				assert.Contains(t, out, want)
			}
			for _, missing := range tc.mustNotHave {
				assert.NotContains(t, out, missing)
			}
			assert.True(t, strings.HasSuffix(out, "\n"))
		})
	}
}

func TestStatementSubmitRejectsMultipleSQLsBeforeWorkspaceClient(t *testing.T) {
	// The "exactly one SQL" check runs in PreRunE BEFORE MustWorkspaceClient,
	// so a malformed invocation is rejected without any auth/profile work.
	// The test relies on this ordering: it does not stub out PreRunE, so if
	// validation moved back after MustWorkspaceClient the test would panic
	// on a missing workspace client instead of returning the validation error.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")
	require.NoError(t, os.WriteFile(path, []byte("SELECT 1"), 0o644))

	cmd := newStatementSubmitCmd()
	cmd.SetArgs([]string{"--file", path, "SELECT 2"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestStatementError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantMsg  string
		wantCode string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "failed with backend error preserves both fields",
			err:      &sqlexec.StatementError{State: sql.StatementStatus_State_Failed, Code: "SYNTAX_ERROR", Message: "near 'bad'"},
			wantMsg:  "near 'bad'",
			wantCode: "SYNTAX_ERROR",
		},
		{
			name:    "failed without backend error surfaces synthesized message",
			err:     &sqlexec.StatementError{State: sql.StatementStatus_State_Failed, Message: "statement reached terminal state FAILED"},
			wantMsg: "statement reached terminal state FAILED",
		},
		{
			name:    "canceled without backend error surfaces synthesized message",
			err:     &sqlexec.StatementError{State: sql.StatementStatus_State_Canceled, Message: "statement reached terminal state CANCELED"},
			wantMsg: "statement reached terminal state CANCELED",
		},
		{
			name:    "closed without backend error surfaces synthesized message",
			err:     &sqlexec.StatementError{State: sql.StatementStatus_State_Closed, Message: "statement reached terminal state CLOSED"},
			wantMsg: "statement reached terminal state CLOSED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statementError(tc.err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantMsg, got.Message)
			assert.Equal(t, tc.wantCode, got.ErrorCode)
		})
	}
}
