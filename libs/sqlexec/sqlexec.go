// Package sqlexec runs SQL statements through the Databricks SQL Statement
// Execution API. It is a general-purpose, non-interactive executor: it submits
// statements, polls them to a terminal state, assembles paginated results, and
// turns failures into typed errors. Programmatic callers such as bundle deploy
// resources (metric views, which have no REST API and are managed via SQL DDL)
// and the experimental aitools query commands share this engine instead of each
// re-implementing the submit/poll/fetch loop.
//
// The engine speaks only the INLINE disposition with the JSON_ARRAY format,
// which the API caps at 25 MiB per result set. That covers every caller today.
// EXTERNAL_LINKS (presigned downloads for larger results, optionally Arrow or
// CSV) is a separate concern and intentionally not implemented here.
//
// A Client holds no mutable state and is safe for concurrent use; aitools fans
// many statements out through a single Client.
package sqlexec

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/service/sql"
)

const (
	// asyncWaitTimeout is the wait applied to Submit. "0s" makes ExecuteStatement
	// return immediately with a statement ID (state PENDING) so callers can wire
	// up cancellation before the statement has a chance to finish.
	asyncWaitTimeout = "0s"

	// defaultWaitTimeout is the synchronous wait Execute applies. Within this
	// window ExecuteStatement blocks server-side, so fast statements (most DDL)
	// return in a single round-trip and never enter the poll loop. The API
	// accepts "0s" or 5s–50s; 10s keeps interactive deploys responsive while
	// absorbing typical warehouse latency.
	defaultWaitTimeout = "10s"

	// defaultPollInterval and defaultPollMax bound the additive backoff Poll
	// applies between GetStatement calls while a statement is PENDING or RUNNING.
	defaultPollInterval = 1 * time.Second
	defaultPollMax      = 5 * time.Second

	// pollIntervalStep is how much the poll interval grows after each poll.
	pollIntervalStep = 1 * time.Second
)

// Client executes SQL statements against a single SQL warehouse.
type Client struct {
	api         sql.StatementExecutionInterface
	warehouseID string

	waitTimeout  string
	pollInterval time.Duration
	pollMax      time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithWaitTimeout sets the synchronous wait Execute applies before falling back
// to polling. Must be "0s" or "5s".."50s" per the API; values outside that range
// are rejected by the backend at submit time.
func WithWaitTimeout(d string) Option {
	return func(c *Client) { c.waitTimeout = d }
}

// WithPollInterval sets the initial and maximum delay between status polls. The
// delay grows additively from initial to max. Tests use a small interval to
// avoid real sleeps.
func WithPollInterval(initial, max time.Duration) Option {
	return func(c *Client) {
		c.pollInterval = initial
		c.pollMax = max
	}
}

// New returns a Client that runs statements on warehouseID via api.
func New(api sql.StatementExecutionInterface, warehouseID string, opts ...Option) *Client {
	c := &Client{
		api:          api,
		warehouseID:  warehouseID,
		waitTimeout:  defaultWaitTimeout,
		pollInterval: defaultPollInterval,
		pollMax:      defaultPollMax,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RequestOption mutates the ExecuteStatementRequest for a single submission.
type RequestOption func(*sql.ExecuteStatementRequest)

// WithParameters binds named parameters (`:name` markers) on the statement.
// Parameter binding is server-side, so values need no manual quoting or
// escaping; prefer it over string interpolation.
func WithParameters(params []sql.StatementParameterListItem) RequestOption {
	return func(req *sql.ExecuteStatementRequest) { req.Parameters = params }
}

// Statement is a handle to a submitted statement and its latest known state.
type Statement struct {
	ID    string
	State sql.StatementState

	// resp is the most recent response observed for the statement. It carries the
	// manifest and first result chunk needed by Results.
	resp *sql.StatementResponse
}

// newStatement wraps a response into a Statement handle.
func newStatement(resp *sql.StatementResponse) *Statement {
	s := &Statement{ID: resp.StatementId, resp: resp}
	if resp.Status != nil {
		s.State = resp.Status.State
	}
	return s
}

// Err returns a *StatementError if the statement is in a terminal non-success
// state (FAILED, CANCELED, CLOSED) or carries no status, and nil otherwise.
// Calling Err on a still-running statement returns nil.
func (s *Statement) Err() error {
	status := s.resp.Status
	if status == nil {
		// The API always populates status; a nil here means a malformed or
		// partial response, which we surface rather than silently treat as empty.
		return &StatementError{Message: "statement response had no status"}
	}
	switch status.State {
	case sql.StatementStateFailed, sql.StatementStateCanceled, sql.StatementStateClosed:
		return newStatementError(status)
	default:
		// SUCCEEDED, PENDING, RUNNING: no error.
		return nil
	}
}

// Columns returns the result column names from the statement's manifest. They
// are known once the statement has succeeded, before any row chunk is fetched,
// so callers can still report column metadata when a later chunk fetch fails.
func (s *Statement) Columns() []string {
	return columns(s.resp.Manifest)
}

// StatementError describes a statement that reached a terminal non-success
// state. FAILED statuses carry a backend error code and message (and, in the
// FAILED case, an SQLSTATE); CANCELED and CLOSED carry no error object, so the
// message is synthesized from the state.
type StatementError struct {
	State    sql.StatementState
	Code     sql.ServiceErrorCode
	Message  string
	SQLState string
}

// newStatementError builds a StatementError from a terminal non-success status.
func newStatementError(status *sql.StatementStatus) *StatementError {
	e := &StatementError{State: status.State, SQLState: status.SqlState}
	if status.Error != nil {
		e.Code = status.Error.ErrorCode
		e.Message = status.Error.Message
	} else {
		e.Message = fmt.Sprintf("statement reached terminal state %s", status.State)
	}
	return e
}

func (e *StatementError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("statement failed: %s: %s", e.Code, e.Message)
	}
	return e.Message
}

// Result is the assembled result set of a statement: column names and every row
// across all chunks. Statements that return no result set (DDL) yield empty
// Columns and Rows.
type Result struct {
	Columns []string
	Rows    [][]string
}

// Scalar returns the top-left cell of the result, or "" when there are no rows.
func (r *Result) Scalar() string {
	if len(r.Rows) == 0 || len(r.Rows[0]) == 0 {
		return ""
	}
	return r.Rows[0][0]
}

// Submit starts a statement asynchronously and returns immediately with its
// handle (state PENDING). Use Poll to wait for completion and Cancel to stop it.
func (c *Client) Submit(ctx context.Context, statement string, opts ...RequestOption) (*Statement, error) {
	return c.submit(ctx, statement, asyncWaitTimeout, opts)
}

// submit issues ExecuteStatement with the given synchronous wait timeout.
func (c *Client) submit(ctx context.Context, statement, waitTimeout string, opts []RequestOption) (*Statement, error) {
	req := sql.ExecuteStatementRequest{
		WarehouseId:   c.warehouseID,
		Statement:     statement,
		WaitTimeout:   waitTimeout,
		OnWaitTimeout: sql.ExecuteStatementRequestOnWaitTimeoutContinue,
		Disposition:   sql.DispositionInline,
		Format:        sql.FormatJsonArray,
	}
	for _, opt := range opts {
		opt(&req)
	}

	resp, err := c.api.ExecuteStatement(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute statement: %w", err)
	}
	return newStatement(resp), nil
}

// Poll waits for a statement to reach a terminal state, returning the updated
// handle. A statement that is already terminal is returned without an API call.
//
// On context cancellation Poll returns the context error WITHOUT cancelling the
// statement server-side; callers that want server-side cancellation must call
// Cancel explicitly. This keeps Poll usable both for "stop watching" (statement
// get) and "stop the query" (interactive query) callers.
func (c *Client) Poll(ctx context.Context, s *Statement) (*Statement, error) {
	interval := c.pollInterval
	for isPending(s.resp.Status) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		resp, err := c.api.GetStatementByStatementId(ctx, s.ID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("poll statement %s: %w", s.ID, err)
		}
		s = newStatement(resp)
		interval = min(interval+pollIntervalStep, c.pollMax)
	}
	return s, nil
}

// Get returns the current state of a statement with a single GET, no polling.
func (c *Client) Get(ctx context.Context, statementID string) (*Statement, error) {
	resp, err := c.api.GetStatementByStatementId(ctx, statementID)
	if err != nil {
		return nil, fmt.Errorf("get statement %s: %w", statementID, err)
	}
	return newStatement(resp), nil
}

// Cancel requests server-side cancellation of a statement. A successful return
// only means the request was accepted; the statement may have already finished.
// Poll or Get to observe the resulting state.
func (c *Client) Cancel(ctx context.Context, statementID string) error {
	if err := c.api.CancelExecution(ctx, sql.CancelExecutionRequest{StatementId: statementID}); err != nil {
		return fmt.Errorf("cancel statement %s: %w", statementID, err)
	}
	return nil
}

// Results assembles the full result set of a statement, fetching every chunk
// beyond the first that the manifest reports. It does not check the statement
// state; call Err first (or use Execute) to reject non-success statements.
func (c *Client) Results(ctx context.Context, s *Statement) (*Result, error) {
	r := &Result{Columns: columns(s.resp.Manifest)}
	if s.resp.Result == nil {
		return r, nil
	}
	r.Rows = append(r.Rows, s.resp.Result.DataArray...)

	total := 0
	if s.resp.Manifest != nil {
		total = s.resp.Manifest.TotalChunkCount
	}
	// Chunk 0 is already in resp.Result; fetch the rest in order.
	for chunk := 1; chunk < total; chunk++ {
		data, err := c.api.GetStatementResultChunkNByStatementIdAndChunkIndex(ctx, s.ID, chunk)
		if err != nil {
			return nil, fmt.Errorf("fetch result chunk %d of statement %s: %w", chunk, s.ID, err)
		}
		r.Rows = append(r.Rows, data.DataArray...)
	}
	return r, nil
}

// Execute submits a statement synchronously, polls it to a terminal state, and
// returns its assembled result. A terminal non-success state is returned as a
// *StatementError.
func (c *Client) Execute(ctx context.Context, statement string, opts ...RequestOption) (*Result, error) {
	s, err := c.submit(ctx, statement, c.waitTimeout, opts)
	if err != nil {
		return nil, err
	}
	s, err = c.Poll(ctx, s)
	if err != nil {
		return nil, err
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return c.Results(ctx, s)
}

// ExecuteScalar runs a statement returning at most one row and one column and
// returns that cell, or "" when there are no rows.
func (c *Client) ExecuteScalar(ctx context.Context, statement string, opts ...RequestOption) (string, error) {
	r, err := c.Execute(ctx, statement, opts...)
	if err != nil {
		return "", err
	}
	return r.Scalar(), nil
}

// isPending reports whether a statement is still PENDING or RUNNING, i.e. Poll
// should keep waiting. A nil status (only possible from a malformed response) is
// not pending, so Poll stops and Err surfaces the missing status to the caller
// rather than looping forever.
func isPending(status *sql.StatementStatus) bool {
	if status == nil {
		return false
	}
	switch status.State {
	case sql.StatementStatePending, sql.StatementStateRunning:
		return true
	default:
		return false
	}
}

// columns returns the column names from a result manifest, or nil when the
// manifest carries no schema (e.g. a statement with no result set).
func columns(manifest *sql.ResultManifest) []string {
	if manifest == nil || manifest.Schema == nil {
		return nil
	}
	out := make([]string, len(manifest.Schema.Columns))
	for i, col := range manifest.Schema.Columns {
		out[i] = col.Name
	}
	return out
}
