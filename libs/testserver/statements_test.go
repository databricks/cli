package testserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/databricks/cli/libs/testserver/testsql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// submitSQL runs a statement through the POST handler and returns the response.
func submitSQL(t *testing.T, server *Server, statement string) *sql.StatementResponse {
	t.Helper()
	body, err := json.Marshal(sql.ExecuteStatementRequest{Statement: statement, WaitTimeout: "10s"})
	require.NoError(t, err)
	resp, ok := server.sqlExecuteStatement(Request{Body: body}).(*sql.StatementResponse)
	require.True(t, ok)
	return resp
}

func TestHandleSQL(t *testing.T) {
	server := New(t)
	server.HandleSQL("SELECT 1 AS a", func(r testsql.Request) testsql.Result {
		assert.Equal(t, []string{"SELECT 1 AS a"}, r.Match)
		return testsql.Result{Columns: []string{"a"}, Rows: [][]string{{"1"}}}
	})

	resp := submitSQL(t, server, "SELECT 1 AS a")
	require.NotNil(t, resp.Status)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	require.NotNil(t, resp.Result)
	assert.Equal(t, [][]string{{"1"}}, resp.Result.DataArray)
}

func TestHandleSQLPattern(t *testing.T) {
	server := New(t)
	// A regex matcher echoes back a captured submatch, exercising both the
	// HandleSQLPattern registration and Request.Match.
	server.HandleSQLPattern(regexp.MustCompile(`^SELECT (\d+)$`), func(r testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"n"}, Rows: [][]string{{r.Match[1]}}}
	})

	resp := submitSQL(t, server, "SELECT 42")
	require.NotNil(t, resp.Status)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	require.NotNil(t, resp.Result)
	assert.Equal(t, [][]string{{"42"}}, resp.Result.DataArray)
}

// postSQL submits a statement over HTTP to path and returns the response.
func postSQL(t *testing.T, server *Server, path, statement string) *sql.StatementResponse {
	t.Helper()
	body, err := json.Marshal(sql.ExecuteStatementRequest{Statement: statement, WarehouseId: "wh-1", WaitTimeout: "10s"})
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out sql.StatementResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return &out
}

func TestDefaultHandlersAcceptFixtureStatements(t *testing.T) {
	server := New(t)
	AddDefaultHandlers(server)

	tests := []struct {
		path      string
		statement string
		want      sql.StatementState
	}{
		{"/api/2.0/sql/statements", "CREATE TABLE main.s.t (id INT)", sql.StatementStateSucceeded},
		{"/api/2.0/sql/statements/", "CREATE TABLE IF NOT EXISTS main.s.t AS SELECT 1", sql.StatementStateSucceeded},
		{"/api/2.0/sql/statements/", "create function main.s.f(x INT) RETURNS INT RETURN x", sql.StatementStateSucceeded},
		{"/api/2.0/sql/statements/", "INSERT INTO main.s.t VALUES (1)", sql.StatementStateSucceeded},
		{"/api/2.0/sql/statements/", "DROP TABLE main.s.t", sql.StatementStateFailed},
	}
	for _, tc := range tests {
		t.Run(tc.statement, func(t *testing.T) {
			resp := postSQL(t, server, tc.path, tc.statement)
			require.NotNil(t, resp.Status)
			assert.Equal(t, tc.want, resp.Status.State)
		})
	}
}

func TestDefaultFixtureMatcherYieldsToTestMatcher(t *testing.T) {
	server := New(t)
	AddDefaultHandlers(server)
	server.HandleSQL("CREATE TABLE main.s.t (id INT)", func(testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"a"}, Rows: [][]string{{"1"}}}
	})

	resp := postSQL(t, server, "/api/2.0/sql/statements", "CREATE TABLE main.s.t (id INT)")
	require.NotNil(t, resp.Result)
	assert.Equal(t, [][]string{{"1"}}, resp.Result.DataArray)
}
