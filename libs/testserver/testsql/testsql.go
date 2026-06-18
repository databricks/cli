// Package testsql is a fake of the Databricks SQL Statement Execution API for
// the testserver. It is NOT a SQL engine: tests register matchers that map a
// submitted statement to a declarative Result (columns, rows, an optional
// error, a poll count, and a chunk count). On submit the handler routes the
// statement to the first matching matcher, runs that matcher exactly once to
// capture a plan, stores the plan under a deterministic statement ID, and then
// replays it across the statement lifecycle (poll -> chunk pagination ->
// cancel).
//
// Because the matcher runs only once per Submit, a matcher may close over
// mutable state (e.g. a map) and have its side effects persist across
// statements, which lets tests model stateful resources (create then read
// back) without a real backend.
package testsql

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/sql"
)

// asyncWaitTimeout is the WaitTimeout value that callers send to submit a
// statement asynchronously. The real API returns immediately with state
// PENDING in this case so the caller can wire up cancellation before the
// statement finishes, so the fake mirrors that and never returns a terminal
// state for "0s" regardless of the matcher's poll count.
const asyncWaitTimeout = "0s"

// Request is what a matcher sees for one submitted statement.
type Request struct {
	// Statement is the trimmed statement text.
	Statement string
	// Match holds regex submatches for a HandlePattern matcher; for an exact
	// Handle matcher it is []string{Statement}.
	Match []string
	// Parameters are the bound :name parameters from the submit request.
	Parameters []sql.StatementParameterListItem
}

// Result is the declarative plan a matcher returns. The zero value is a
// terminal SUCCEEDED statement with no columns and no rows in a single inline
// chunk.
type Result struct {
	Columns []string
	Rows    [][]string
	// Error, when non-nil, makes the statement terminate as FAILED (which the
	// real API reports over HTTP 200, not an HTTP error).
	Error *Error
	// Polls is the number of non-terminal GET responses returned before the
	// statement reaches a terminal state.
	Polls int
	// Chunks splits Rows across this many chunk fetches. 0 or 1 means a single
	// inline chunk.
	Chunks int
}

// Error describes a terminal FAILED statement.
type Error struct {
	Code     sql.ServiceErrorCode
	Message  string
	SQLState string
}

// matcher pairs a predicate against a statement with the plan-producing fn.
type matcher struct {
	// match returns the submatches to pass to fn, or nil when the statement
	// does not match this matcher.
	match func(stmt string) []string
	fn    func(Request) Result
}

// statement is the captured plan for one submitted statement, replayed across
// its lifecycle.
type statement struct {
	id      string
	columns []string
	// chunks holds the result split into fetchable chunks. A statement with rows
	// has at least one chunk (chunk 0 carries the inline data returned with the
	// terminal response); a statement with no rows has zero chunks.
	chunks         [][][]string
	err            *Error
	remainingPolls int
	canceled       bool
}

// Handler routes submitted statements to matchers and replays their captured
// plans across the statement lifecycle.
type Handler struct {
	mu         sync.Mutex
	matchers   []matcher
	statements map[string]*statement
	nextID     int
}

// New returns a Handler with no matchers registered.
func New() *Handler {
	return &Handler{statements: map[string]*statement{}}
}

// Handle registers a matcher that runs fn when a submitted statement equals
// statement exactly (after trimming). The first registered matching matcher
// wins. The matcher's Request.Match is []string{statement}.
func (h *Handler) Handle(statement string, fn func(Request) Result) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.matchers = append(h.matchers, matcher{
		match: func(stmt string) []string {
			if stmt == statement {
				return []string{stmt}
			}
			return nil
		},
		fn: fn,
	})
}

// HandlePattern registers a matcher that runs fn when re matches a submitted
// statement. The first registered matching matcher wins. The matcher's
// Request.Match is re.FindStringSubmatch(stmt).
func (h *Handler) HandlePattern(re *regexp.Regexp, fn func(Request) Result) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.matchers = append(h.matchers, matcher{
		match: func(stmt string) []string {
			return re.FindStringSubmatch(stmt)
		},
		fn: fn,
	})
}

// Submit matches the statement, runs the matcher once to capture its plan,
// stores it under a new deterministic ID ("statement-1", "statement-2", ...),
// and returns the first lifecycle response. A "0s" wait timeout always yields
// PENDING; otherwise the response is terminal when the plan has no remaining
// polls and PENDING otherwise.
func (h *Handler) Submit(statement, waitTimeout string, params []sql.StatementParameterListItem) *sql.StatementResponse {
	h.mu.Lock()
	defer h.mu.Unlock()

	stmt := h.plan(strings.TrimSpace(statement), params)
	h.nextID++
	stmt.id = fmt.Sprintf("statement-%d", h.nextID)
	h.statements[stmt.id] = stmt

	if waitTimeout == asyncWaitTimeout || stmt.remainingPolls > 0 {
		return &sql.StatementResponse{
			StatementId: stmt.id,
			Status:      &sql.StatementStatus{State: sql.StatementStatePending},
		}
	}
	return stmt.terminalResponse()
}

// Get returns the current state of the statement. A canceled statement reports
// CANCELED (this takes precedence over remaining polls). Otherwise a statement
// with remaining polls decrements its counter and reports RUNNING; a statement
// with no remaining polls reports its terminal state with manifest and chunk 0.
// An unknown id returns nil.
func (h *Handler) Get(statementID string) *sql.StatementResponse {
	h.mu.Lock()
	defer h.mu.Unlock()

	stmt := h.statements[statementID]
	if stmt == nil {
		return nil
	}
	if stmt.canceled {
		return &sql.StatementResponse{
			StatementId: stmt.id,
			Status:      &sql.StatementStatus{State: sql.StatementStateCanceled},
		}
	}
	if stmt.remainingPolls > 0 {
		stmt.remainingPolls--
		return &sql.StatementResponse{
			StatementId: stmt.id,
			Status:      &sql.StatementStatus{State: sql.StatementStateRunning},
		}
	}
	return stmt.terminalResponse()
}

// Chunk returns the rows of chunk chunkIndex for the statement. An unknown id
// or out-of-range index returns nil.
func (h *Handler) Chunk(statementID string, chunkIndex int) *sql.ResultData {
	h.mu.Lock()
	defer h.mu.Unlock()

	stmt := h.statements[statementID]
	if stmt == nil || chunkIndex < 0 || chunkIndex >= len(stmt.chunks) {
		return nil
	}
	return &sql.ResultData{DataArray: stmt.chunks[chunkIndex]}
}

// Cancel marks the statement canceled. An unknown id is a no-op.
func (h *Handler) Cancel(statementID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if stmt := h.statements[statementID]; stmt != nil {
		stmt.canceled = true
	}
}

// plan runs the first matching matcher once and turns its Result into a stored
// statement. With no matching matcher the statement terminates as FAILED with
// an "unsupported statement" error, mirroring how the real backend rejects a
// statement it cannot run.
func (h *Handler) plan(stmt string, params []sql.StatementParameterListItem) *statement {
	res := Result{Error: &Error{
		Code:    sql.ServiceErrorCodeBadRequest,
		Message: "unsupported statement: " + stmt,
	}}
	for _, m := range h.matchers {
		groups := m.match(stmt)
		if groups == nil {
			continue
		}
		res = m.fn(Request{Statement: stmt, Match: groups, Parameters: params})
		break
	}
	return &statement{
		columns:        res.Columns,
		chunks:         splitChunks(res.Rows, res.Chunks),
		err:            res.Error,
		remainingPolls: res.Polls,
	}
}

// terminalResponse builds the terminal response for a statement: FAILED when it
// carries an error, otherwise SUCCEEDED with the manifest and inline chunk 0.
func (s *statement) terminalResponse() *sql.StatementResponse {
	if s.err != nil {
		return &sql.StatementResponse{
			StatementId: s.id,
			Status: &sql.StatementStatus{
				State:    sql.StatementStateFailed,
				SqlState: s.err.SQLState,
				Error:    &sql.ServiceError{ErrorCode: s.err.Code, Message: s.err.Message},
			},
		}
	}

	manifest := &sql.ResultManifest{TotalChunkCount: len(s.chunks)}
	if len(s.columns) > 0 {
		cols := make([]sql.ColumnInfo, len(s.columns))
		for i, name := range s.columns {
			cols[i] = sql.ColumnInfo{Name: name}
		}
		manifest.Schema = &sql.ResultSchema{Columns: cols}
	}
	// A no-row statement reports an empty result (matching the real API, which
	// returns "result": {} with no data_array); otherwise chunk 0 is inlined.
	result := &sql.ResultData{}
	if len(s.chunks) > 0 {
		result.DataArray = s.chunks[0]
	}
	return &sql.StatementResponse{
		StatementId: s.id,
		Status:      &sql.StatementStatus{State: sql.StatementStateSucceeded},
		Manifest:    manifest,
		Result:      result,
	}
}

// splitChunks divides rows into max(1, chunks) chunks as evenly as possible by
// ceil division. A statement with no rows yields zero chunks (TotalChunkCount=0
// and an empty result), matching the real API's response to a 0-row SELECT or a
// no-result-set DDL statement.
func splitChunks(rows [][]string, chunks int) [][][]string {
	if len(rows) == 0 {
		return nil
	}
	n := max(1, chunks)
	out := make([][][]string, n)
	size := (len(rows) + n - 1) / n
	for i := range n {
		start := min(i*size, len(rows))
		end := min(start+size, len(rows))
		out[i] = rows[start:end]
	}
	return out
}
