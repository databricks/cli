package sqlexec_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/testserver/testsql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive the engine through a real SDK client over HTTP against the
// in-process testserver, with the statement-execution endpoints programmed per
// test via server.HandleSQL matchers. Unlike the mock-interface unit tests they
// exercise the full request/response JSON serialization, and unlike the
// integration tests they are hermetic and run on every PR without a warehouse.
func httpClient(t *testing.T, server *testserver.Server) *sqlexec.Client {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	// Tiny poll interval so the polling tests don't sleep.
	return sqlexec.New(w.StatementExecution, "wh-1", sqlexec.WithPollInterval(time.Millisecond, time.Millisecond))
}

func TestHTTPExecuteSuccess(t *testing.T) {
	server := testserver.New(t)
	server.HandleSQL("SELECT 1 AS a, 2 AS b", func(testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"a", "b"}, Rows: [][]string{{"1", "2"}}}
	})

	r, err := httpClient(t, server).Execute(t.Context(), "SELECT 1 AS a, 2 AS b")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, r.Columns)
	assert.Equal(t, [][]string{{"1", "2"}}, r.Rows)
}

func TestHTTPExecutePolls(t *testing.T) {
	server := testserver.New(t)
	server.HandleSQL("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{Rows: [][]string{{"done"}}, Polls: 1}
	})

	got, err := httpClient(t, server).ExecuteScalar(t.Context(), "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "done", got)
}

func TestHTTPExecutePaginatesChunks(t *testing.T) {
	server := testserver.New(t)
	server.HandleSQL("SELECT * FROM big", func(testsql.Request) testsql.Result {
		return testsql.Result{Rows: [][]string{{"0"}, {"1"}, {"2"}}, Chunks: 3}
	})

	r, err := httpClient(t, server).Execute(t.Context(), "SELECT * FROM big")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"0"}, {"1"}, {"2"}}, r.Rows)
}

func TestHTTPExecuteFailedReturns200(t *testing.T) {
	server := testserver.New(t)
	// A failed statement comes back as HTTP 200 with state=FAILED, not an HTTP
	// error; the engine must inspect the body and surface a *StatementError.
	server.HandleSQL("SELECT * FROM nope", func(testsql.Request) testsql.Result {
		return testsql.Result{Error: &testsql.Error{Code: sql.ServiceErrorCodeBadRequest, Message: "no such table", SQLState: "42P01"}}
	})

	_, err := httpClient(t, server).Execute(t.Context(), "SELECT * FROM nope")
	var se *sqlexec.StatementError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, sql.StatementStateFailed, se.State)
	assert.Equal(t, sql.ServiceErrorCodeBadRequest, se.Code)
	assert.Equal(t, "no such table", se.Message)
	assert.Equal(t, "42P01", se.SQLState)
}

func TestHTTPSubmitAndCancel(t *testing.T) {
	server := testserver.New(t)
	server.HandleSQL("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{}
	})

	c := httpClient(t, server)
	ctx := t.Context()

	stmt, err := c.Submit(ctx, "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "statement-1", stmt.ID)

	require.NoError(t, c.Cancel(ctx, stmt.ID))

	stmt, err = c.Poll(ctx, stmt)
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateCanceled, stmt.State)
	require.Error(t, stmt.Err())
}
