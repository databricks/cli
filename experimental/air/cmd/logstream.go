package aircmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// Log-stream tuning constants. These mirror the Python CLI (log_streaming.py) so
// behavior stays identical after the port.
const (
	// retryCheckInterval is how long the poll loop waits between status/log polls.
	retryCheckInterval = 3 * time.Second
	// maxTransientFailures is how many consecutive Bricklens failures we tolerate
	// before treating it as unavailable and falling back to MLflow.
	maxTransientFailures = 5
	// defaultCompletedRunTailLines caps a completed run's output when neither
	// --lines nor --minutes is set, so a multi-chunk log does not flood stdout.
	defaultCompletedRunTailLines = 10000
	// seenRecordsCap bounds the dedup set so a large initial drain does not
	// accumulate every record; oldest-inserted entries are evicted first.
	seenRecordsCap = 100000
)

// errBricklensFeatureDisabled signals the caller to fall back to the MLflow log
// path. It is returned when Bricklens can't serve logs: the endpoint is gated
// off by a backend SAFE flag (FEATURE_DISABLED), not deployed (ENDPOINT_NOT_FOUND
// / 404), or persistently failing after maxTransientFailures retries. The flag is
// evaluated server-side; the CLI only reads the resulting error code.
var errBricklensFeatureDisabled = errors.New("bricklens logs unavailable; falling back to mlflow")

// logRequest is the resolved, backend-agnostic description of what to fetch. It
// is shared by the Bricklens streamer and the MLflow fallback so both honor the
// same flags. windowMinutes and tailLines are mutually exclusive (validated at
// the command layer): windowMinutes drives the time window, tailLines the tail.
type logRequest struct {
	runID int64
	// node is the node index to fetch; 0 by default (node 0 always exists).
	node int
	// attempt is the retry attempt to read; -1 means latest.
	attempt int
	// windowMinutes, when > 0, restricts the fetch to the last N minutes.
	windowMinutes int
	// tailLines, when > 0, keeps only the last N lines of a completed run.
	tailLines int
	// staticView renders a one-shot tail instead of following the run. It is set
	// for a past retry of a still-active run: that attempt's logs are immutable,
	// so streaming them would poll forever waiting for the run (not the attempt)
	// to finish. A completed run is inherently static and does not need this.
	staticView bool
	jsonOutput bool
}

// runStatus is the subset of a run's state the log path needs. It is resolved
// once from a Jobs GetRun and reused, avoiding a per-tick typed-vs-dict shuffle.
type logRunStatus struct {
	lifeCycleState string
	resultState    string
	stateMessage   string
	startTimeMs    int64
	endTimeMs      int64
	// latestAttempt is the highest attempt_number across the run's tasks.
	latestAttempt int
}

// terminalLifeCycleStates and terminalResultStates classify a finished run. A
// run is terminal when its lifecycle state is terminal, or a result state is set
// (result states only appear on terminal runs). Mirrors log_streaming.py.
var (
	terminalLifeCycleStates = map[string]bool{"TERMINATED": true, "SKIPPED": true, "INTERNAL_ERROR": true}
	terminalResultStates    = map[string]bool{"SUCCESS": true, "FAILED": true, "CANCELED": true}
)

func (s logRunStatus) terminal() bool {
	return terminalLifeCycleStates[s.lifeCycleState] || terminalResultStates[s.resultState]
}

func (s logRunStatus) succeeded() bool {
	return s.resultState == "SUCCESS"
}

// resolveRunStatus fetches a run's state via the Jobs API and projects it onto
// logRunStatus. The run id being unknown surfaces as apierr.ErrResourceDoesNotExist
// so the caller can report a clean not-found.
func resolveRunStatus(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (logRunStatus, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return logRunStatus{}, err
	}
	return projectRunStatus(run), nil
}

// projectRunStatus extracts logRunStatus from a Jobs run. Split out from
// resolveRunStatus so it can be unit-tested without an API client.
func projectRunStatus(run *jobs.Run) logRunStatus {
	s := logRunStatus{
		startTimeMs: run.StartTime,
		endTimeMs:   run.EndTime,
	}
	if run.State != nil {
		s.lifeCycleState = string(run.State.LifeCycleState)
		s.resultState = string(run.State.ResultState)
		s.stateMessage = run.State.StateMessage
	}
	for i := range run.Tasks {
		s.latestAttempt = max(s.latestAttempt, run.Tasks[i].AttemptNumber)
	}
	return s
}

// classifyLogError decides how a Bricklens failure should be handled:
//   - errBricklensFeatureDisabled: fall back to MLflow (flag gated off, endpoint
//     absent, or a 404 — older logs may still live in MLflow).
//   - the original error: a genuine not-found, surfaced as-is.
//   - nil: a transient failure the caller should retry.
//
// The backend evaluates the SAFE flag; this only reads the returned error code,
// so it never string-matches error text (repo rule).
func classifyLogError(err error) error {
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		switch apiErr.ErrorCode {
		case "FEATURE_DISABLED", "ENDPOINT_NOT_FOUND":
			return errBricklensFeatureDisabled
		}
		if apiErr.StatusCode == http.StatusNotFound {
			return errBricklensFeatureDisabled
		}
	}
	if errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return err
	}
	return nil
}

// fromSeconds computes the Bricklens `from` bound for a request. With --minutes
// set it is now-N*60; otherwise it is the run's start second (0 when the run has
// not started, which the endpoint reads as "everything stored"). Matches
// stream_logs_via_bricklens.
func (req logRequest) fromSeconds(status logRunStatus, now time.Time) int64 {
	if req.windowMinutes > 0 {
		return now.Add(-time.Duration(req.windowMinutes) * time.Minute).Unix()
	}
	if status.startTimeMs > 0 {
		return status.startTimeMs / 1000
	}
	return 0
}

// toSeconds computes the Bricklens `to` bound. Once a run is terminal no new logs
// can appear, so we cap at the run's end second (ceil of the millisecond end time
// so records in the final partial second aren't excluded); otherwise 0 lets the
// endpoint default to now.
func (req logRequest) toSeconds(status logRunStatus) int64 {
	if status.terminal() && status.endTimeMs > 0 {
		return (status.endTimeMs + 999) / 1000
	}
	return 0
}

// streamBricklensLogs fetches and prints a run's logs via Bricklens, handling
// both an already-completed run (a single bounded tail drain) and an active run
// (a poll-and-drain loop that follows until the run reaches a terminal state).
//
// It returns (success, err) where success reports whether the run finished with
// SUCCESS. A returned errBricklensFeatureDisabled means the caller should fall
// back to the MLflow path; the run genuinely not existing is returned as-is.
func streamBricklensLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	st := &bricklensStreamer{
		ctx:    ctx,
		w:      w,
		out:    out,
		req:    req,
		status: status,
		seen:   newSeenSet(seenRecordsCap),
	}
	return st.run()
}

// bricklensStreamer carries the streaming state across the poll loop: the running
// from-second cursor, the highest emitted timestamp, and the bounded dedup set.
type bricklensStreamer struct {
	ctx    context.Context
	w      *databricks.WorkspaceClient
	out    io.Writer
	req    logRequest
	status logRunStatus

	fromSec      int64
	lastNano     int64
	firstLogSeen bool
	seen         *seenSet
}

func (st *bricklensStreamer) run() (bool, error) {
	now := time.Now()
	st.fromSec = st.req.fromSeconds(st.status, now)

	// A past retry's logs are immutable, so render them as a one-shot tail even
	// when the run is still active — the attempt has ended, and following the run
	// would poll forever. Mirrors handle_logs' viewing_past_retry.
	if st.req.staticView {
		return st.drainStatic(st.req.toSeconds(st.status))
	}

	firstIteration := true
	for {
		if !firstIteration {
			status, err := resolveRunStatus(st.ctx, st.w, st.req.runID)
			if err != nil {
				if errors.Is(err, apierr.ErrResourceDoesNotExist) {
					return false, err
				}
				// A transient status blip should not abort a live stream; log and retry.
				log.Debugf(st.ctx, "air logs: failed to refresh run status: %v", err)
				time.Sleep(retryCheckInterval)
				continue
			}
			st.status = status
		}

		terminal := st.status.terminal()
		toSec := st.req.toSeconds(st.status)

		// An already-completed run (terminal on the first iteration) renders as a
		// tail: only the most-recent N lines, oldest-first. An active run — even
		// one that terminates while we watch — streams everything and dedups so the
		// final drain doesn't re-print the boundary second. A tail is line-based, so
		// it only applies to the completed-run case (matching the MLflow path).
		var err error
		if firstIteration && terminal {
			err = st.drainTail(toSec)
		} else {
			err = st.drainPages(toSec)
		}
		if err != nil {
			return false, err
		}

		if terminal {
			if !st.firstLogSeen {
				st.emitNoLogs()
			}
			log.Infof(st.ctx, "air logs: run %d finished in state %s", st.req.runID, st.status.displayState())
			return st.status.succeeded(), nil
		}

		firstIteration = false
		time.Sleep(retryCheckInterval)
	}
}

// drainStatic renders a single tail pass for an immutable attempt and returns
// without following the run. Success reflects the run's current result state
// (empty for an active run, so a past retry of a running job is not reported as
// a failure).
func (st *bricklensStreamer) drainStatic(toSec int64) (bool, error) {
	if err := st.drainTail(toSec); err != nil {
		return false, err
	}
	if !st.firstLogSeen {
		st.emitNoLogs()
	}
	return st.status.succeeded(), nil
}

// tailTarget is the number of lines a completed-run tail should keep: --lines
// when set, else the default cap.
func (req logRequest) tailTarget() int {
	if req.tailLines > 0 {
		return req.tailLines
	}
	return defaultCompletedRunTailLines
}

// drainTail emits the most-recent `target` records for a completed run, oldest-first.
// Bricklens returns records newest-first, so it pages until it has at least
// `target`, keeps the newest `target`, and reverses to chronological order —
// matching the MLflow --tail behavior. No dedup: a single bounded pass whose
// ordering we own client-side.
func (st *bricklensStreamer) drainTail(toSec int64) error {
	target := st.req.tailTarget()
	if target <= 0 {
		return nil
	}

	var collected []logRecord
	var pageToken string
	for len(collected) < target {
		resp, err := st.requestPage(pageToken, toSec, target, false)
		if err != nil {
			return err
		}
		collected = append(collected, resp.LogRecords...)
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	// Keep the newest `target` (returned newest-first), then reverse so they print
	// oldest -> newest like the MLflow tail.
	if len(collected) > target {
		collected = collected[:target]
	}
	for _, c := range slices.Backward(collected) {
		st.emit(c.Body)
	}
	return nil
}

// drainPages exhausts all available pages from the current from-second in
// ascending (oldest-first) order so a live run streams chronologically,
// deduping against the bounded seen-set so a re-queried boundary second is not
// re-printed. It advances fromSec to the floor-second of the newest record seen.
func (st *bricklensStreamer) drainPages(toSec int64) error {
	var pageToken string
	for {
		resp, err := st.requestPage(pageToken, toSec, 0, true)
		if err != nil {
			return err
		}

		for _, rec := range resp.LogRecords {
			nano := rec.nano()
			if nano != 0 {
				// Ascending fetch, so a record older than the last emitted one is
				// out of order (or a re-queried boundary record); skip it to keep
				// streamed output monotonic.
				if st.lastNano != 0 && nano < st.lastNano {
					continue
				}
				if st.seen.has(nano, rec.Body) {
					continue
				}
			}
			st.emit(rec.Body)
			if nano != 0 {
				st.seen.add(nano, rec.Body)
				st.lastNano = max(st.lastNano, nano)
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	if st.lastNano != 0 {
		st.fromSec = st.lastNano / 1_000_000_000
	}
	return nil
}

// requestPage fetches one page, applying the fallback classification. It retries
// transient failures up to maxTransientFailures, then falls back like a disabled
// feature (older logs may still be in MLflow). A feature-gated response or a
// genuine not-found returns immediately.
func (st *bricklensStreamer) requestPage(pageToken string, toSec int64, pageSize int, ascending bool) (*bricklensLogsResponse, error) {
	q := bricklensLogsQuery{
		fromSeconds:   st.fromSec,
		toSeconds:     toSec,
		pageToken:     pageToken,
		pageSize:      pageSize,
		attemptNumber: st.req.attempt,
		nodeIndex:     st.req.node,
		ascending:     ascending,
	}

	transientFailures := 0
	for {
		resp, err := getBricklensLogs(st.ctx, st.w, st.req.runID, q)
		if err == nil {
			return resp, nil
		}

		switch classified := classifyLogError(err); {
		case errors.Is(classified, errBricklensFeatureDisabled):
			return nil, errBricklensFeatureDisabled
		case classified != nil:
			return nil, classified
		}

		transientFailures++
		if transientFailures >= maxTransientFailures {
			log.Debugf(st.ctx, "air logs: get_logs failed %d times; falling back to mlflow", maxTransientFailures)
			return nil, errBricklensFeatureDisabled
		}
		log.Debugf(st.ctx, "air logs: get_logs transient failure (%d/%d): %v", transientFailures, maxTransientFailures, err)
		time.Sleep(retryCheckInterval)
	}
}

// emit writes one log line and latches firstLogSeen so a terminal run with no
// output can report "no logs".
func (st *bricklensStreamer) emit(body string) {
	st.firstLogSeen = true
	emitLogLine(st.out, st.req, body)
}

// emitNoLogs reports that a terminal run produced no logs.
func (st *bricklensStreamer) emitNoLogs() {
	emitNoLogs(st.out, st.req, st.status)
}

// displayState is the run's terminal result state, falling back to the lifecycle
// state, else "UNKNOWN".
func (s logRunStatus) displayState() string {
	if s.resultState != "" {
		return s.resultState
	}
	if s.lifeCycleState != "" {
		return s.lifeCycleState
	}
	return "UNKNOWN"
}
