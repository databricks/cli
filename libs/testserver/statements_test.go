package testserver

import (
	"encoding/json"
	statementexecution "github.com/databricks/cli/libs/sdktypes/statementexecution/v1"
	"github.com/databricks/cli/libs/utils"
	"regexp"
	"testing"

	"github.com/databricks/cli/libs/testserver/testsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// submitSQL runs a statement through the POST handler and returns the response.
func submitSQL(t *testing.T, server *Server, statement string) *statementexecution.StatementResponse {
	t.Helper()
	body, err := json.Marshal(statementexecution.ExecuteStatementRequest{Statement: utils.NonZero(statement), WaitTimeout: utils.NonZero("10s")})
	require.NoError(t, err)
	resp, ok := server.sqlExecuteStatement(Request{Body: body}).(*statementexecution.StatementResponse)
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
	assert.Equal(t, statementexecution.StatementStatus_State_Succeeded, resp.Status.State)
	require.NotNil(t, resp.Result)
	assert.Equal(t, [][]string{{"1"}}, sqlRows(resp.Result.DataArray))
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
	assert.Equal(t, statementexecution.StatementStatus_State_Succeeded, resp.Status.State)
	require.NotNil(t, resp.Result)
	assert.Equal(t, [][]string{{"42"}}, sqlRows(resp.Result.DataArray))
}

func sqlRows(data [][]json.RawMessage) [][]string {
	out := make([][]string, len(data))
	for i, row := range data {
		out[i] = make([]string, len(row))
		for j, cell := range row {
			_ = json.Unmarshal(cell, &out[i][j])
		}
	}
	return out
}
