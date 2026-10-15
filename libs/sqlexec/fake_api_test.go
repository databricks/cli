package sqlexec

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/call"
	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/mock"
)

// mockAPI is a testify mock of API. Calls are recorded by the identifiers the
// executor passes (statement ID, chunk index) so expectations stay readable.
type mockAPI struct {
	mock.Mock
}

func newMockAPI(t interface {
	mock.TestingT
	Cleanup(func())
},
) *mockAPI {
	m := &mockAPI{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *mockAPI) ExecuteStatement(ctx context.Context, req sql.ExecuteStatementRequest, _ ...call.Option) (*sql.StatementResponse, error) {
	args := m.Called(ctx, req)
	resp, _ := args.Get(0).(*sql.StatementResponse)
	return resp, args.Error(1)
}

func (m *mockAPI) GetStatementResult(ctx context.Context, req sql.GetStatementResultRequest, _ ...call.Option) (*sql.StatementResponse, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId))
	resp, _ := args.Get(0).(*sql.StatementResponse)
	return resp, args.Error(1)
}

func (m *mockAPI) GetResultData(ctx context.Context, req sql.GetResultDataRequest, _ ...call.Option) (*sql.ResultData, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId), utils.Deref(req.ChunkIndex))
	resp, _ := args.Get(0).(*sql.ResultData)
	return resp, args.Error(1)
}

func (m *mockAPI) CancelStatement(ctx context.Context, req sql.CancelStatementRequest, _ ...call.Option) (*sql.CancelStatementResponse, error) {
	args := m.Called(ctx, utils.Deref(req.StatementId))
	return &sql.CancelStatementResponse{}, args.Error(0)
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
