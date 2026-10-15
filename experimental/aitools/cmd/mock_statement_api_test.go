package aitools

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/call"
	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/mock"
)

// mockStatementAPI is a testify mock of sqlexec.API. Expectations match on the
// request (ExecuteStatement) or on the statement ID and chunk index.
type mockStatementAPI struct {
	mock.Mock
}

func newMockStatementAPI(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockStatementAPI {
	m := &mockStatementAPI{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *mockStatementAPI) ExecuteStatement(ctx context.Context, req sql.ExecuteStatementRequest, _ ...call.Option) (*sql.StatementResponse, error) {
	args := m.Called(ctx, req)
	if f, ok := args.Get(0).(func(context.Context, sql.ExecuteStatementRequest) (*sql.StatementResponse, error)); ok {
		return f(ctx, req)
	}
	resp, _ := args.Get(0).(*sql.StatementResponse)
	return resp, args.Error(1)
}

func (m *mockStatementAPI) GetStatementResult(ctx context.Context, req sql.GetStatementResultRequest, _ ...call.Option) (*sql.StatementResponse, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId))
	resp, _ := args.Get(0).(*sql.StatementResponse)
	return resp, args.Error(1)
}

func (m *mockStatementAPI) GetResultData(ctx context.Context, req sql.GetResultDataRequest, _ ...call.Option) (*sql.ResultData, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId), utils.Deref(req.ChunkIndex))
	resp, _ := args.Get(0).(*sql.ResultData)
	return resp, args.Error(1)
}

func (m *mockStatementAPI) CancelStatement(ctx context.Context, req sql.CancelStatementRequest, _ ...call.Option) (*sql.CancelStatementResponse, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId))
	return &sql.CancelStatementResponse{}, args.Error(0)
}

func (m *mockStatementAPI) EXPECT() mockStatementAPIExpecter {
	return mockStatementAPIExpecter{m: &m.Mock}
}

type mockStatementAPIExpecter struct {
	m *mock.Mock
}

func (e mockStatementAPIExpecter) ExecuteStatement(ctx, req any) executeStatementCall {
	return executeStatementCall{e.m.On("ExecuteStatement", ctx, req)}
}

func (e mockStatementAPIExpecter) GetStatementResult(ctx, statementID any) *mock.Call {
	return e.m.On("GetStatementResult", ctx, statementID)
}

func (e mockStatementAPIExpecter) GetResultData(ctx, statementID, chunkIndex any) *mock.Call {
	return e.m.On("GetResultData", ctx, statementID, chunkIndex)
}

func (e mockStatementAPIExpecter) CancelStatement(ctx, statementID any) *mock.Call {
	return e.m.On("CancelStatement", ctx, statementID)
}

type executeStatementCall struct {
	*mock.Call
}

// RunAndReturn answers each matching call with f.
func (c executeStatementCall) RunAndReturn(f func(context.Context, sql.ExecuteStatementRequest) (*sql.StatementResponse, error)) *mock.Call {
	return c.Return(f, nil)
}

// rows encodes cells the way the API returns JSON_ARRAY results.
func rows(data [][]string) [][]json.RawMessage {
	if data == nil {
		return nil
	}
	out := make([][]json.RawMessage, len(data))
	for i, row := range data {
		out[i] = make([]json.RawMessage, len(row))
		for j, cell := range row {
			out[i][j], _ = json.Marshal(cell)
		}
	}
	return out
}
