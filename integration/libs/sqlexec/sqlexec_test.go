package sqlexec_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/sqlexec"
	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise libs/sqlexec against a live SQL warehouse to confirm the
// engine keeps working against the real Statement Execution API. They run in the
// nightly integration suite and skip without CLOUD_ENV / TEST_DEFAULT_WAREHOUSE_ID.
//
// They are limited to UC workspaces: gating on TEST_METASTORE_ID (set only on the
// *-ucws environments) keeps them off the non-UC workspaces, whose shared classic
// warehouses are unreliable to start. CLOUD_ENV can't make this distinction
// because azure-prod-ucws and the non-UC azure-prod both report CLOUD_ENV=azure.
func newClient(t *testing.T) (*acc.WorkspaceT, *sqlexec.Client) {
	t.Helper()
	testutil.GetEnvOrSkipTest(t, "TEST_METASTORE_ID")
	_, wt := acc.WorkspaceTest(t)
	warehouseID := testutil.GetEnvOrSkipTest(t, "TEST_DEFAULT_WAREHOUSE_ID")
	return wt, sqlexec.New(wt.W.StatementExecution, warehouseID)
}

func TestSQLExecScalar(t *testing.T) {
	wt, c := newClient(t)
	got, err := c.ExecuteScalar(wt.Context(), "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

func TestSQLExecColumnsAndRows(t *testing.T) {
	wt, c := newClient(t)
	r, err := c.Execute(wt.Context(), "SELECT id, id * 2 AS doubled FROM range(3) ORDER BY id")
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "doubled"}, r.Columns)
	assert.Equal(t, [][]string{{"0", "0"}, {"1", "2"}, {"2", "4"}}, r.Rows)
}

func TestSQLExecParameters(t *testing.T) {
	wt, c := newClient(t)
	r, err := c.Execute(wt.Context(), "SELECT :n AS n, :s AS s", sqlexec.WithParameters([]sql.StatementParameter{
		{Name: new("n"), Type: new("INT"), Value: new("42")},
		{Name: new("s"), Value: new("hello")},
	}))
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"42", "hello"}}, r.Rows)
}

func TestSQLExecNullParameter(t *testing.T) {
	wt, c := newClient(t)
	// An empty value is sent as SQL NULL (StatementParameterListItem.Value is omitempty).
	got, err := c.ExecuteScalar(wt.Context(), "SELECT :maybe IS NULL", sqlexec.WithParameters([]sql.StatementParameter{
		{Name: new("maybe"), Value: new("")},
	}))
	require.NoError(t, err)
	assert.Equal(t, "true", got)
}

func TestSQLExecFailedStatement(t *testing.T) {
	wt, c := newClient(t)
	_, err := c.Execute(wt.Context(), "SELECT * FROM a_table_that_does_not_exist_zzz")
	var se *sqlexec.StatementError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, sql.StatementStatus_State_Failed, se.State)
	assert.NotEmpty(t, se.Code)
	assert.NotEmpty(t, se.Message)
	// The API populates status.sql_state (SQLSTATE) on failures, e.g. 42P01 for
	// a missing relation; assert it survives into the typed error.
	assert.NotEmpty(t, se.SQLState)
}

func TestSQLExecSubmitAndCancel(t *testing.T) {
	wt, c := newClient(t)
	ctx := wt.Context()
	// Submit returns immediately with the statement ID so we can cancel it.
	stmt, err := c.Submit(ctx, "SELECT count(*) FROM range(100000000000)")
	require.NoError(t, err)
	require.NotEmpty(t, stmt.ID)

	require.NoError(t, c.Cancel(ctx, stmt.ID))

	// The cancel is best-effort; poll to a terminal state and accept either a
	// CANCELED statement or one that finished before the cancel landed.
	stmt, err = c.Poll(ctx, stmt)
	require.NoError(t, err)
	if stmt.State == sql.StatementStatus_State_Canceled {
		require.Error(t, stmt.Err())
		var se *sqlexec.StatementError
		require.ErrorAs(t, stmt.Err(), &se)
		assert.Equal(t, sql.StatementStatus_State_Canceled, se.State)
	} else {
		assert.Equal(t, sql.StatementStatus_State_Succeeded, stmt.State)
	}
}
