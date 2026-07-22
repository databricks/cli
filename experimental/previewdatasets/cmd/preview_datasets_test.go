package previewdatasetscmd

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	mocksql "github.com/databricks/databricks-sdk-go/experimental/mocks/service/sql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveWarehouseIDFlagWins(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{Config: &config.Config{WarehouseID: "from-config"}}
	id, err := resolveWarehouseID(ctx, w, "from-flag")
	require.NoError(t, err)
	assert.Equal(t, "from-flag", id)
}

func TestResolveWarehouseIDFallsBackToConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{Config: &config.Config{WarehouseID: "from-config"}}
	id, err := resolveWarehouseID(ctx, w, "")
	require.NoError(t, err)
	assert.Equal(t, "from-config", id)
}

func TestResolveWarehouseIDUsesDefaultWarehouse(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockWH := mocksql.NewMockWarehousesInterface(t)
	mockWH.EXPECT().Get(mock.Anything, sql.GetWarehouseRequest{Id: "default"}).
		Return(&sql.GetWarehouseResponse{Id: "wh-default"}, nil)

	w := &databricks.WorkspaceClient{Config: &config.Config{}, Warehouses: mockWH}
	id, err := resolveWarehouseID(ctx, w, "")
	require.NoError(t, err)
	assert.Equal(t, "wh-default", id)
}

func TestExecuteSelectQuotesAndLimits(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.MatchedBy(func(req sql.ExecuteStatementRequest) bool {
		return req.WarehouseId == "wh-1" &&
			req.Statement == "SELECT * FROM `main`.`sales`.`orders` LIMIT 25"
	})).Return(&sql.StatementResponse{
		StatementId: "s1",
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "id"}}}},
		Result:      &sql.ResultData{DataArray: [][]string{{"1"}}},
	}, nil)

	quoted, err := quoteTableName("main.sales.orders")
	require.NoError(t, err)
	client := sqlexec.New(mockAPI, "wh-1")
	result, err := client.Execute(ctx, "SELECT * FROM "+quoted+" LIMIT 25")
	require.NoError(t, err)
	assert.Equal(t, []string{"id"}, result.Columns)
	assert.Equal(t, [][]string{{"1"}}, result.Rows)
}

func TestExecuteSelectSurfacesStatementError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	mockAPI := mocksql.NewMockStatementExecutionInterface(t)

	mockAPI.EXPECT().ExecuteStatement(mock.Anything, mock.Anything).Return(&sql.StatementResponse{
		StatementId: "s1",
		Status: &sql.StatementStatus{
			State: sql.StatementStateFailed,
			Error: &sql.ServiceError{ErrorCode: sql.ServiceErrorCodeBadRequest, Message: "TABLE_OR_VIEW_NOT_FOUND"},
		},
	}, nil)

	client := sqlexec.New(mockAPI, "wh-1")
	_, err := client.Execute(ctx, "SELECT * FROM `x`.`y`.`z` LIMIT 25")
	require.Error(t, err)

	var stmtErr *sqlexec.StatementError
	require.ErrorAs(t, err, &stmtErr)
	assert.Equal(t, sql.ServiceErrorCodeBadRequest, stmtErr.Code)
}
