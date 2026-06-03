package sqlexec_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/sqlexec"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive the engine through a real SDK client over HTTP against the
// in-process testserver, with the statement-execution endpoints programmed per
// test. Unlike the mock-interface unit tests they exercise the full
// request/response JSON serialization, and unlike the integration tests they are
// hermetic and run on every PR without a warehouse.
func httpClient(t *testing.T, server *testserver.Server) *sqlexec.Client {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	// Tiny poll interval so the polling tests don't sleep.
	return sqlexec.New(w.StatementExecution, "wh-1", sqlexec.WithPollInterval(time.Millisecond, time.Millisecond))
}

func TestHTTPExecuteSuccess(t *testing.T) {
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/sql/statements", func(testserver.Request) any {
		return sql.StatementResponse{
			StatementId: "s1",
			Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
			Manifest:    &sql.ResultManifest{Schema: &sql.ResultSchema{Columns: []sql.ColumnInfo{{Name: "a"}, {Name: "b"}}}, TotalChunkCount: 1},
			Result:      &sql.ResultData{DataArray: [][]string{{"1", "2"}}},
		}
	})

	r, err := httpClient(t, server).Execute(t.Context(), "SELECT 1 AS a, 2 AS b")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, r.Columns)
	assert.Equal(t, [][]string{{"1", "2"}}, r.Rows)
}

func TestHTTPExecutePolls(t *testing.T) {
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/sql/statements", func(testserver.Request) any {
		return sql.StatementResponse{StatementId: "s1", Status: &sql.StatementStatus{State: sql.StatementStatePending}}
	})
	polls := 0
	server.Handle("GET", "/api/2.0/sql/statements/{statement_id}", func(req testserver.Request) any {
		assert.Equal(t, "s1", req.Vars["statement_id"])
		polls++
		if polls < 2 {
			return sql.StatementResponse{StatementId: "s1", Status: &sql.StatementStatus{State: sql.StatementStateRunning}}
		}
		return sql.StatementResponse{
			StatementId: "s1",
			Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
			Result:      &sql.ResultData{DataArray: [][]string{{"done"}}},
		}
	})

	got, err := httpClient(t, server).ExecuteScalar(t.Context(), "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "done", got)
	assert.GreaterOrEqual(t, polls, 2)
}

func TestHTTPExecutePaginatesChunks(t *testing.T) {
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/sql/statements", func(testserver.Request) any {
		return sql.StatementResponse{
			StatementId: "s1",
			Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
			Manifest:    &sql.ResultManifest{TotalChunkCount: 3},
			Result:      &sql.ResultData{DataArray: [][]string{{"0"}}},
		}
	})
	server.Handle("GET", "/api/2.0/sql/statements/{statement_id}/result/chunks/{chunk_index}", func(req testserver.Request) any {
		return sql.ResultData{DataArray: [][]string{{req.Vars["chunk_index"]}}}
	})

	r, err := httpClient(t, server).Execute(t.Context(), "SELECT * FROM big")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"0"}, {"1"}, {"2"}}, r.Rows)
}

func TestHTTPExecuteFailedReturns200(t *testing.T) {
	server := testserver.New(t)
	// A failed statement comes back as HTTP 200 with state=FAILED, not an HTTP
	// error; the engine must inspect the body and surface a *StatementError.
	server.Handle("POST", "/api/2.0/sql/statements", func(testserver.Request) any {
		return sql.StatementResponse{
			StatementId: "s1",
			Status: &sql.StatementStatus{
				State:    sql.StatementStateFailed,
				SqlState: "42P01",
				Error:    &sql.ServiceError{ErrorCode: sql.ServiceErrorCodeBadRequest, Message: "no such table"},
			},
		}
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
	server.Handle("POST", "/api/2.0/sql/statements", func(testserver.Request) any {
		return sql.StatementResponse{StatementId: "s1", Status: &sql.StatementStatus{State: sql.StatementStatePending}}
	})
	canceled := false
	server.Handle("POST", "/api/2.0/sql/statements/{statement_id}/cancel", func(req testserver.Request) any {
		assert.Equal(t, "s1", req.Vars["statement_id"])
		canceled = true
		return map[string]string{}
	})
	server.Handle("GET", "/api/2.0/sql/statements/{statement_id}", func(testserver.Request) any {
		state := sql.StatementStatePending
		if canceled {
			state = sql.StatementStateCanceled
		}
		return sql.StatementResponse{StatementId: "s1", Status: &sql.StatementStatus{State: state}}
	})

	c := httpClient(t, server)
	ctx := t.Context()

	stmt, err := c.Submit(ctx, "SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "s1", stmt.ID)

	require.NoError(t, c.Cancel(ctx, stmt.ID))
	assert.True(t, canceled)

	stmt, err = c.Poll(ctx, stmt)
	require.NoError(t, err)
	assert.Equal(t, sql.StatementStateCanceled, stmt.State)
	require.Error(t, stmt.Err())
}
