package testsql_test

import (
	"regexp"
	"testing"

	"github.com/databricks/cli/libs/testserver/testsql"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExactDispatch(t *testing.T) {
	e := testsql.New()
	var seen testsql.Request
	e.Handle("SELECT 1", func(r testsql.Request) testsql.Result {
		seen = r
		return testsql.Result{Columns: []string{"a"}, Rows: [][]string{{"1"}}}
	})

	// Statement is trimmed before matching.
	resp := e.Submit("  SELECT 1  ", "10s", nil)
	require.NotNil(t, resp.Status)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	assert.Equal(t, "SELECT 1", seen.Statement)
	assert.Equal(t, []string{"SELECT 1"}, seen.Match)
	assert.Equal(t, []string{"a"}, columnNames(resp))
	assert.Equal(t, [][]string{{"1"}}, resp.Result.DataArray)
}

func TestPatternDispatchSubmatches(t *testing.T) {
	e := testsql.New()
	re := regexp.MustCompile(`^CREATE TABLE (\w+)$`)
	var seen testsql.Request
	e.HandlePattern(re, func(r testsql.Request) testsql.Result {
		seen = r
		return testsql.Result{}
	})

	resp := e.Submit("CREATE TABLE foo", "10s", nil)
	assert.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	assert.Equal(t, []string{"CREATE TABLE foo", "foo"}, seen.Match)
}

func TestFirstMatchWins(t *testing.T) {
	e := testsql.New()
	e.HandlePattern(regexp.MustCompile(`SELECT`), func(testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"first"}}
	})
	e.Handle("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"second"}}
	})

	resp := e.Submit("SELECT 1", "10s", nil)
	assert.Equal(t, []string{"first"}, columnNames(resp))
}

func TestUnsupportedStatement(t *testing.T) {
	e := testsql.New()
	resp := e.Submit("SELECT 1", "10s", nil)
	require.NotNil(t, resp.Status)
	assert.Equal(t, sql.StatementStateFailed, resp.Status.State)
	require.NotNil(t, resp.Status.Error)
	assert.Equal(t, sql.ServiceErrorCodeBadRequest, resp.Status.Error.ErrorCode)
	assert.Contains(t, resp.Status.Error.Message, "unsupported statement")
	assert.Contains(t, resp.Status.Error.Message, "SELECT 1")
}

func TestWaitTimeout(t *testing.T) {
	tests := []struct {
		name        string
		waitTimeout string
		want        sql.StatementState
	}{
		{"async stays pending even without polls", "0s", sql.StatementStatePending},
		{"sync with no polls terminates", "10s", sql.StatementStateSucceeded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := testsql.New()
			e.Handle("SELECT 1", func(testsql.Request) testsql.Result {
				return testsql.Result{Rows: [][]string{{"x"}}}
			})
			resp := e.Submit("SELECT 1", tc.waitTimeout, nil)
			assert.Equal(t, tc.want, resp.Status.State)
		})
	}
}

func TestPollCountdown(t *testing.T) {
	e := testsql.New()
	e.Handle("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{Rows: [][]string{{"done"}}, Polls: 2}
	})

	resp := e.Submit("SELECT 1", "10s", nil)
	require.Equal(t, sql.StatementStatePending, resp.Status.State)

	got := e.Get(resp.StatementId)
	assert.Equal(t, sql.StatementStateRunning, got.Status.State)
	got = e.Get(resp.StatementId)
	assert.Equal(t, sql.StatementStateRunning, got.Status.State)

	got = e.Get(resp.StatementId)
	require.Equal(t, sql.StatementStateSucceeded, got.Status.State)
	assert.Equal(t, [][]string{{"done"}}, got.Result.DataArray)
}

func TestChunkSplitting(t *testing.T) {
	e := testsql.New()
	rows := [][]string{{"0"}, {"1"}, {"2"}}
	e.Handle("SELECT * FROM big", func(testsql.Request) testsql.Result {
		return testsql.Result{Rows: rows, Chunks: 3}
	})

	resp := e.Submit("SELECT * FROM big", "10s", nil)
	require.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	require.NotNil(t, resp.Manifest)
	assert.Equal(t, 3, resp.Manifest.TotalChunkCount)

	var all [][]string
	all = append(all, resp.Result.DataArray...)
	for i := 1; i < resp.Manifest.TotalChunkCount; i++ {
		chunk := e.Chunk(resp.StatementId, i)
		require.NotNil(t, chunk)
		all = append(all, chunk.DataArray...)
	}
	assert.Equal(t, rows, all)
	assert.Equal(t, [][]string{{"0"}}, resp.Result.DataArray)
	assert.Equal(t, [][]string{{"1"}}, e.Chunk(resp.StatementId, 1).DataArray)
	assert.Equal(t, [][]string{{"2"}}, e.Chunk(resp.StatementId, 2).DataArray)
}

func TestNoRowsZeroChunks(t *testing.T) {
	e := testsql.New()
	e.Handle("CREATE TABLE foo", func(testsql.Request) testsql.Result {
		return testsql.Result{}
	})

	// A no-row statement reports zero chunks and an empty result, matching the
	// real API (a 0-row SELECT or a no-result-set DDL returns total_chunk_count=0).
	resp := e.Submit("CREATE TABLE foo", "10s", nil)
	require.Equal(t, sql.StatementStateSucceeded, resp.Status.State)
	require.NotNil(t, resp.Manifest)
	assert.Equal(t, 0, resp.Manifest.TotalChunkCount)
	assert.Nil(t, resp.Manifest.Schema)
	require.NotNil(t, resp.Result)
	assert.Empty(t, resp.Result.DataArray)
}

func TestErrorResult(t *testing.T) {
	e := testsql.New()
	e.Handle("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{Error: &testsql.Error{
			Code:     sql.ServiceErrorCodeBadRequest,
			Message:  "no such table",
			SQLState: "42P01",
		}}
	})

	resp := e.Submit("SELECT 1", "10s", nil)
	require.Equal(t, sql.StatementStateFailed, resp.Status.State)
	assert.Equal(t, "42P01", resp.Status.SqlState)
	require.NotNil(t, resp.Status.Error)
	assert.Equal(t, sql.ServiceErrorCodeBadRequest, resp.Status.Error.ErrorCode)
	assert.Equal(t, "no such table", resp.Status.Error.Message)
	assert.Nil(t, resp.Manifest)
	assert.Nil(t, resp.Result)
}

func TestCancelPrecedence(t *testing.T) {
	e := testsql.New()
	e.Handle("SELECT 1", func(testsql.Request) testsql.Result {
		return testsql.Result{Polls: 5}
	})

	resp := e.Submit("SELECT 1", "0s", nil)
	require.Equal(t, sql.StatementStatePending, resp.Status.State)

	e.Cancel(resp.StatementId)

	// Cancel takes precedence over the remaining polls.
	got := e.Get(resp.StatementId)
	assert.Equal(t, sql.StatementStateCanceled, got.Status.State)
	assert.Nil(t, got.Manifest)
	assert.Nil(t, got.Result)
}

func TestUnknownStatement(t *testing.T) {
	e := testsql.New()
	assert.Nil(t, e.Get("statement-99"))
	assert.Nil(t, e.Chunk("statement-99", 0))
	e.Cancel("statement-99") // no-op, must not panic
}

func TestDeterministicIDs(t *testing.T) {
	e := testsql.New()
	e.Handle("SELECT 1", func(testsql.Request) testsql.Result { return testsql.Result{} })
	assert.Equal(t, "statement-1", e.Submit("SELECT 1", "10s", nil).StatementId)
	assert.Equal(t, "statement-2", e.Submit("SELECT 1", "10s", nil).StatementId)
}

// TestStatefulMatcher proves the matcher runs exactly once per Submit and that
// side effects persist across statements: a create matcher stores a body keyed
// by a regex submatch, and a separate select matcher reads it back.
func TestStatefulMatcher(t *testing.T) {
	e := testsql.New()
	store := map[string]string{}
	calls := 0

	e.HandlePattern(regexp.MustCompile(`^CREATE (\w+) AS (.+)$`), func(r testsql.Request) testsql.Result {
		calls++
		store[r.Match[1]] = r.Match[2]
		return testsql.Result{}
	})
	e.HandlePattern(regexp.MustCompile(`^READ (\w+)$`), func(r testsql.Request) testsql.Result {
		return testsql.Result{Columns: []string{"def"}, Rows: [][]string{{store[r.Match[1]]}}}
	})

	create := e.Submit("CREATE v AS body", "10s", nil)
	require.Equal(t, sql.StatementStateSucceeded, create.Status.State)
	assert.Equal(t, 1, calls, "matcher must run exactly once per Submit")

	read := e.Submit("READ v", "10s", nil)
	require.Equal(t, sql.StatementStateSucceeded, read.Status.State)
	assert.Equal(t, [][]string{{"body"}}, read.Result.DataArray)
	assert.Equal(t, 1, calls, "create matcher must not run again")
}

// columnNames mirrors how libs/sqlexec reads column names off the manifest.
func columnNames(resp *sql.StatementResponse) []string {
	if resp.Manifest == nil || resp.Manifest.Schema == nil {
		return nil
	}
	out := make([]string, len(resp.Manifest.Schema.Columns))
	for i, c := range resp.Manifest.Schema.Columns {
		out[i] = c.Name
	}
	return out
}
