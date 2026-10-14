package pipelines

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/options/call"
	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWarehouseIDFlagWins(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &sdkclients.Clients{Config: &config.Config{WarehouseID: "from-config"}}
	id, err := resolveWarehouseID(ctx, w, "from-flag")
	require.NoError(t, err)
	assert.Equal(t, "from-flag", id)
}

func TestResolveWarehouseIDFallsBackToConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &sdkclients.Clients{Config: &config.Config{WarehouseID: "from-config"}}
	id, err := resolveWarehouseID(ctx, w, "")
	require.NoError(t, err)
	assert.Equal(t, "from-config", id)
}

func TestResolveWarehouseIDUsesDefaultWarehouse(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cfg, _ := httpfixtures.Fixtures{{
		Method:   "GET",
		Resource: "/api/2.0/sql/warehouses/default",
		Response: `{"id":"wh-default"}`,
	}}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)
	id, err := resolveWarehouseID(ctx, w, "")
	require.NoError(t, err)
	assert.Equal(t, "wh-default", id)
}

// fakeStatements implements sqlexec.API; only ExecuteStatement is expected.
type fakeStatements struct {
	execute func(context.Context, sql.ExecuteStatementRequest) (*sql.StatementResponse, error)
}

func (f fakeStatements) ExecuteStatement(ctx context.Context, req sql.ExecuteStatementRequest, _ ...call.Option) (*sql.StatementResponse, error) {
	return f.execute(ctx, req)
}

func (fakeStatements) GetStatementResult(context.Context, sql.GetStatementResultRequest, ...call.Option) (*sql.StatementResponse, error) {
	return nil, errors.New("unexpected GetStatementResult")
}

func (fakeStatements) GetResultData(context.Context, sql.GetResultDataRequest, ...call.Option) (*sql.ResultData, error) {
	return nil, errors.New("unexpected GetResultData")
}

func (fakeStatements) CancelStatement(context.Context, sql.CancelStatementRequest, ...call.Option) (*sql.CancelStatementResponse, error) {
	return &sql.CancelStatementResponse{}, nil
}

func TestExecuteSelectQuotesAndLimits(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	api := fakeStatements{execute: func(_ context.Context, req sql.ExecuteStatementRequest) (*sql.StatementResponse, error) {
		assert.Equal(t, "wh-1", utils.Deref(req.WarehouseId))
		assert.Equal(t, "SELECT * FROM `main`.`sales`.`orders` LIMIT 25", utils.Deref(req.Statement))
		return &sql.StatementResponse{
			StatementId: new("s1"),
			Status:      &sql.StatementStatus{State: sql.StatementStatus_State_Succeeded},
			Manifest:    &sql.ResultManifest{Schema: &sql.Schema{Columns: []sql.ColumnInfo{{Name: new("id")}}}},
			Result:      &sql.ResultData{DataArray: [][]json.RawMessage{{json.RawMessage(`"1"`)}}},
		}, nil
	}}

	quoted, err := quoteTableName("main.sales.orders")
	require.NoError(t, err)
	client := sqlexec.New(api, "wh-1")
	result, err := client.Execute(ctx, "SELECT * FROM "+quoted+" LIMIT 25")
	require.NoError(t, err)
	assert.Equal(t, []string{"id"}, result.Columns)
	assert.Equal(t, [][]string{{"1"}}, result.Rows)
}

func TestExecuteSelectSurfacesStatementError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	api := fakeStatements{execute: func(context.Context, sql.ExecuteStatementRequest) (*sql.StatementResponse, error) {
		return &sql.StatementResponse{
			StatementId: new("s1"),
			Status: &sql.StatementStatus{
				State: sql.StatementStatus_State_Failed,
				Error: &sql.ServiceError{ErrorCode: sql.ServiceErrorCode_BadRequest, Message: new("TABLE_OR_VIEW_NOT_FOUND")},
			},
		}, nil
	}}

	client := sqlexec.New(api, "wh-1")
	_, err := client.Execute(ctx, "SELECT * FROM `x`.`y`.`z` LIMIT 25")
	require.Error(t, err)

	var stmtErr *sqlexec.StatementError
	require.ErrorAs(t, err, &stmtErr)
	assert.Equal(t, sql.ServiceErrorCode_BadRequest, stmtErr.Code)
}

func TestExecuteSelectInterruptedSetsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	api := fakeStatements{execute: func(context.Context, sql.ExecuteStatementRequest) (*sql.StatementResponse, error) {
		cancel()
		return &sql.StatementResponse{StatementId: new("s1"), Status: &sql.StatementStatus{State: sql.StatementStatus_State_Pending}}, nil
	}}

	client := sqlexec.New(api, "wh-1")
	_, err := client.Execute(ctx, "SELECT * FROM `x`.`y`.`z` LIMIT 25")
	require.Error(t, err)
	assert.Error(t, ctx.Err())
}
