package sqlexec_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/databricks-sdk-go/service/sql"
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

func TestSQLExecNullParameter(t *testing.T) {
	wt, c := newClient(t)
	// An empty value is sent as SQL NULL (StatementParameterListItem.Value is omitempty).
	got, err := c.ExecuteScalar(wt.Context(), "SELECT :maybe IS NULL", sqlexec.WithParameters([]sql.StatementParameterListItem{
		{Name: "maybe", Value: ""},
	}))
	require.NoError(t, err)
	assert.Equal(t, "true", got)
}

func TestSQLExecFailedStatement(t *testing.T) {
	wt, c := newClient(t)
	_, err := c.Execute(wt.Context(), "SELECT * FROM a_table_that_does_not_exist_zzz")
	var se *sqlexec.StatementError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, sql.StatementStateFailed, se.State)
	assert.NotEmpty(t, se.Code)
	assert.NotEmpty(t, se.Message)
	// The API populates status.sql_state (SQLSTATE) on failures, e.g. 42P01 for
	// a missing relation; assert it survives into the typed error.
	assert.NotEmpty(t, se.SQLState)
}
