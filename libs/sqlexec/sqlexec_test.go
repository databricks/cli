package sqlexec

import (
	"context"
	"errors"
	"testing"
	"time"

	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testClient returns a Client wired to a fresh mock with a near-zero poll
// interval so polling tests don't sleep.
func testClient(t *testing.T) (*Client, *mockAPI) {
	api := newMockAPI(t)
	c := New(api, "wh-1", WithPollInterval(time.Millisecond, time.Millisecond))
	return c, api
}

func statusResp(state sql.StatementStatus_State) *sql.StatementResponse {
	return &sql.StatementResponse{StatementId: new("stmt-1"), Status: &sql.StatementStatus{State: state}}
}

func succeededResp(columns []string, dataArray [][]string) *sql.StatementResponse {
	var schema *sql.Schema
	if columns != nil {
		cols := make([]sql.ColumnInfo, len(columns))
		for i, name := range columns {
			cols[i] = sql.ColumnInfo{Name: new(name)}
		}
		schema = &sql.Schema{Columns: cols}
	}
	return &sql.StatementResponse{
		StatementId: new("stmt-1"),
		Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Succeeded},
		Manifest:    &sql.ResultManifest{Schema: schema},
		Result:      &sql.ResultData{DataArray: rows(dataArray)},
	}
}

func TestExecuteScalar(t *testing.T) {
	tests := []struct {
		name      string
		resp      *sql.StatementResponse
		want      string
		errSubstr string
	}{
		{
			name: "succeeded with cell",
			resp: succeededResp([]string{"table_type"}, [][]string{{"METRIC_VIEW"}}),
			want: "METRIC_VIEW",
		},
		{
			name: "succeeded with no rows",
			resp: succeededResp([]string{"table_type"}, nil),
			want: "",
		},
		{
			name: "failed surfaces backend code and message",
			resp: &sql.StatementResponse{
				StatementId: new("stmt-1"),
				Status: &sql.StatementStatus{
					State:    sql.StatementStatus_State_Failed,
					SqlState: new("42000"),
					Error:    &sql.ServiceError{ErrorCode: sql.ServiceErrorCode_BadRequest, Message: new("boom")},
				},
			},
			errSubstr: "boom",
		},
		{
			name:      "canceled is not treated as success",
			resp:      statusResp(sql.StatementStatus_State_Canceled),
			errSubstr: "terminal state CANCELED",
		},
		{
			name:      "closed is not treated as success",
			resp:      statusResp(sql.StatementStatus_State_Closed),
			errSubstr: "terminal state CLOSED",
		},
		{
			name:      "missing status surfaces as error",
			resp:      &sql.StatementResponse{StatementId: new("stmt-1")},
			errSubstr: "no status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, api := testClient(t)
			api.On("ExecuteStatement", mock.Anything, mock.Anything).Return(tt.resp, nil).Once()

			got, err := c.ExecuteScalar(t.Context(), "SELECT 1")
			if tt.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExecuteDDLReturnsEmptyResult(t *testing.T) {
	c, api := testClient(t)
	// DDL responses carry a status but no manifest or result block.
	api.On("ExecuteStatement", mock.Anything, mock.Anything).
		Return(statusResp(sql.StatementStatus_State_Succeeded), nil).Once()

	r, err := c.Execute(t.Context(), "CREATE VIEW v")
	require.NoError(t, err)
	assert.Empty(t, r.Columns)
	assert.Empty(t, r.Rows)
	assert.Empty(t, r.Scalar())
}

func TestExecuteSubmitTransportError(t *testing.T) {
	c, api := testClient(t)
	api.On("ExecuteStatement", mock.Anything, mock.Anything).
		Return(nil, errors.New("network down")).Once()

	_, err := c.Execute(t.Context(), "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute statement")
	assert.Contains(t, err.Error(), "network down")
}

func TestExecutePollsUntilTerminal(t *testing.T) {
	c, api := testClient(t)
	api.On("ExecuteStatement", mock.Anything, mock.Anything).
		Return(statusResp(sql.StatementStatus_State_Pending), nil).Once()
	api.On("GetStatementResult", mock.Anything, "stmt-1").
		Return(statusResp(sql.StatementStatus_State_Running), nil).Once()
	api.On("GetStatementResult", mock.Anything, "stmt-1").
		Return(succeededResp([]string{"n"}, [][]string{{"polled"}}), nil).Once()

	got, err := c.ExecuteScalar(t.Context(), "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "polled", got)
}

func TestExecuteUsesSyncWaitTimeout(t *testing.T) {
	api := newMockAPI(t)
	c := New(api, "wh-1", WithWaitTimeout("25s"))
	api.On("ExecuteStatement", mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return *req.WaitTimeout == "25s" &&
			req.OnWaitTimeout == sql.TimeoutAction_Continue &&
			req.Disposition == sql.Disposition_Inline &&
			req.Format == sql.Format_JsonArray &&
			*req.WarehouseId == "wh-1"
	})).Return(succeededResp(nil, nil), nil).Once()

	_, err := c.Execute(t.Context(), "SELECT 1")
	require.NoError(t, err)
}

func TestSubmitIsAsyncAndForwardsParameters(t *testing.T) {
	c, api := testClient(t)
	params := []sql.StatementParameter{{Name: new("since"), Type: new("DATE"), Value: new("2026-01-01")}}

	api.On("ExecuteStatement", mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return *req.WaitTimeout == "0s" &&
			req.OnWaitTimeout == sql.TimeoutAction_Continue &&
			assert.ObjectsAreEqual(params, req.Parameters)
	})).Return(statusResp(sql.StatementStatus_State_Pending), nil).Once()

	s, err := c.Submit(t.Context(), "SELECT :since", WithParameters(params))
	require.NoError(t, err)
	assert.Equal(t, "stmt-1", s.ID)
	assert.Equal(t, sql.StatementStatus_State_Pending, s.State)
}

func TestPollImmediateTerminalDoesNotCallAPI(t *testing.T) {
	c, _ := testClient(t)
	// No GetStatement expectation: a terminal statement must not be polled.
	s, err := c.Poll(t.Context(), newStatement(succeededResp(nil, nil)))
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStatus_State_Succeeded, s.State)
}

func TestPollContextCancellationDoesNotCancelServerSide(t *testing.T) {
	c, api := testClient(t)
	_ = api // no CancelExecution expectation: Poll must not cancel server-side.

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.Poll(ctx, newStatement(statusResp(sql.StatementStatus_State_Pending)))
	require.ErrorIs(t, err, context.Canceled)
}

func TestPollWrapsGetError(t *testing.T) {
	c, api := testClient(t)
	api.On("GetStatementResult", mock.Anything, "stmt-1").
		Return(nil, errors.New("boom")).Once()

	_, err := c.Poll(t.Context(), newStatement(statusResp(sql.StatementStatus_State_Pending)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "poll statement stmt-1")
	assert.Contains(t, err.Error(), "boom")
}

func TestPollGetErrorAfterCancellationReturnsContextErr(t *testing.T) {
	api := newMockAPI(t)
	c := New(api, "wh-1", WithPollInterval(time.Millisecond, time.Millisecond))

	ctx, cancel := context.WithCancel(t.Context())
	// Simulate the context being cancelled while the GET is in flight: the call
	// fails, but Poll must surface the context error, not the transport error.
	api.On("GetStatementResult", mock.Anything, "stmt-1").
		Run(func(mock.Arguments) { cancel() }).
		Return(nil, errors.New("cancelled mid-flight")).Once()

	_, err := c.Poll(ctx, newStatement(statusResp(sql.StatementStatus_State_Pending)))
	require.ErrorIs(t, err, context.Canceled)
}

func TestExecuteReturnsPollError(t *testing.T) {
	c, api := testClient(t)
	api.On("ExecuteStatement", mock.Anything, mock.Anything).
		Return(statusResp(sql.StatementStatus_State_Pending), nil).Once()
	api.On("GetStatementResult", mock.Anything, "stmt-1").
		Return(nil, errors.New("boom")).Once()

	_, err := c.Execute(t.Context(), "SELECT 1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "poll statement stmt-1")
}

func TestResultsPaginatesChunks(t *testing.T) {
	c, api := testClient(t)
	resp := succeededResp([]string{"n"}, [][]string{{"0"}})
	resp.Manifest.TotalChunkCount = new(3)
	api.On("GetResultData", mock.Anything, "stmt-1", 1).
		Return(&sql.ResultData{DataArray: rows([][]string{{"1"}})}, nil).Once()
	api.On("GetResultData", mock.Anything, "stmt-1", 2).
		Return(&sql.ResultData{DataArray: rows([][]string{{"2"}})}, nil).Once()

	r, err := c.Results(t.Context(), newStatement(resp))
	require.NoError(t, err)
	assert.Equal(t, []string{"n"}, r.Columns)
	assert.Equal(t, [][]string{{"0"}, {"1"}, {"2"}}, r.Rows)
}

func TestResultsChunkFetchError(t *testing.T) {
	c, api := testClient(t)
	resp := succeededResp([]string{"n"}, [][]string{{"0"}})
	resp.Manifest.TotalChunkCount = new(2)
	api.On("GetResultData", mock.Anything, "stmt-1", 1).
		Return(nil, errors.New("throttled")).Once()

	_, err := c.Results(t.Context(), newStatement(resp))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch result chunk 1 of statement stmt-1")
	assert.Contains(t, err.Error(), "throttled")
}

func TestResultsNoResultBlock(t *testing.T) {
	c, _ := testClient(t)
	r, err := c.Results(t.Context(), newStatement(statusResp(sql.StatementStatus_State_Succeeded)))
	require.NoError(t, err)
	assert.Nil(t, r.Columns)
	assert.Nil(t, r.Rows)
}

func TestGet(t *testing.T) {
	c, api := testClient(t)
	api.On("GetStatementResult", mock.Anything, "stmt-9").
		Return(statusResp(sql.StatementStatus_State_Running), nil).Once()

	s, err := c.Get(t.Context(), "stmt-9")
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStatus_State_Running, s.State)
}

func TestGetWrapsError(t *testing.T) {
	c, api := testClient(t)
	api.On("GetStatementResult", mock.Anything, "stmt-9").
		Return(nil, errors.New("nope")).Once()

	_, err := c.Get(t.Context(), "stmt-9")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get statement stmt-9")
}

func TestCancel(t *testing.T) {
	c, api := testClient(t)
	api.On("CancelStatement", mock.Anything, "stmt-1").
		Return(nil).Once()
	require.NoError(t, c.Cancel(t.Context(), "stmt-1"))
}

func TestCancelWrapsError(t *testing.T) {
	c, api := testClient(t)
	api.On("CancelStatement", mock.Anything, mock.Anything).Return(errors.New("gone")).Once()
	err := c.Cancel(t.Context(), "stmt-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel statement stmt-1")
	assert.Contains(t, err.Error(), "gone")
}

func TestStatementErr(t *testing.T) {
	tests := []struct {
		name     string
		resp     *sql.StatementResponse
		wantErr  bool
		wantCode sql.ServiceErrorCode
		wantMsg  string
	}{
		{name: "succeeded", resp: statusResp(sql.StatementStatus_State_Succeeded)},
		{name: "pending", resp: statusResp(sql.StatementStatus_State_Pending)},
		{name: "running", resp: statusResp(sql.StatementStatus_State_Running)},
		{
			name: "failed with service error",
			resp: &sql.StatementResponse{Status: &sql.StatementStatus{
				State:    sql.StatementStatus_State_Failed,
				SqlState: new("42000"),
				Error:    &sql.ServiceError{ErrorCode: sql.ServiceErrorCode_BadRequest, Message: new("bad")},
			}},
			wantErr:  true,
			wantCode: sql.ServiceErrorCode_BadRequest,
			wantMsg:  "bad",
		},
		{
			name:    "canceled without error object",
			resp:    statusResp(sql.StatementStatus_State_Canceled),
			wantErr: true,
			wantMsg: "statement reached terminal state CANCELED",
		},
		{
			name:    "nil status",
			resp:    &sql.StatementResponse{},
			wantErr: true,
			wantMsg: "statement response had no status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newStatement(tt.resp).Err()
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var se *StatementError
			require.ErrorAs(t, err, &se)
			assert.Equal(t, tt.wantCode, se.Code)
			assert.Equal(t, tt.wantMsg, se.Message)
		})
	}
}

func TestStatementErrorError(t *testing.T) {
	withCode := &StatementError{Code: sql.ServiceErrorCode_BadRequest, Message: "bad"}
	assert.Equal(t, "statement failed: BAD_REQUEST: bad", withCode.Error())

	noCode := &StatementError{State: sql.StatementStatus_State_Canceled, Message: "statement reached terminal state CANCELED"}
	assert.Equal(t, "statement reached terminal state CANCELED", noCode.Error())
}

func TestResultScalar(t *testing.T) {
	assert.Empty(t, (&Result{}).Scalar())
	assert.Empty(t, (&Result{Rows: [][]string{{}}}).Scalar())
	assert.Equal(t, "x", (&Result{Rows: [][]string{{"x", "y"}}}).Scalar())
}

func TestStatementColumns(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, newStatement(succeededResp([]string{"a", "b"}, nil)).Columns())
	// No manifest (e.g. a DDL response): no columns.
	assert.Nil(t, newStatement(statusResp(sql.StatementStatus_State_Succeeded)).Columns())
}
