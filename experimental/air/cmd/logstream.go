package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const (
	// maxTransientFailures is how many consecutive Bricklens failures to tolerate
	// before falling back to MLflow.
	maxTransientFailures = 5
	// defaultCompletedRunTailLines caps a completed run's output when neither
	// --lines nor --minutes is set.
	defaultCompletedRunTailLines = 10000
	// seenRecordsCap bounds the dedup set, evicting oldest-inserted entries first.
	seenRecordsCap = 100000
)

// retryCheckInterval is the wait between status/log polls. A var so tests can
// shrink it.
var retryCheckInterval = 3 * time.Second

// errBricklensFeatureDisabled signals the caller to fall back to MLflow: Bricklens
// is gated off (FEATURE_DISABLED), not deployed (ENDPOINT_NOT_FOUND / 404), or
// persistently failing. The flag is evaluated server-side.
var errBricklensFeatureDisabled = errors.New("bricklens logs unavailable; falling back to mlflow")

// logRequest describes what to fetch, shared by both backends so they honor the
// same flags. windowMinutes and tailLines are mutually exclusive.
type logRequest struct {
	runID int64
	// node is the node index to fetch; node 0 always exists.
	node int
	// nodeSet distinguishes an explicit --node 0 from the default, so a download
	// knows whether to fetch one node or all of them.
	nodeSet bool
	// attempt is the retry attempt to read; -1 means latest.
	attempt int
	// windowMinutes, when > 0, restricts the fetch to the last N minutes.
	windowMinutes int
	// tailLines caps a completed run's output to the last N lines. Negative means
	// --lines was unset (use the default cap); 0 prints nothing.
	tailLines int
	// downloadTo, when set, writes logs to that directory instead of stdout.
	downloadTo string
	// staticView renders a one-shot tail instead of following the run. Set for a
	// past retry of an active run: that attempt's logs are immutable, so streaming
	// would poll forever waiting for the run (not the attempt) to finish.
	staticView bool
	jsonOutput bool
	// onStatusChange, when set, is called on each lifecycle transition while
	// following the run (current, previous display states). Used by
	// `air run --watch -o json` to emit STATUS events.
	onStatusChange func(current, previous string)
}

// logRunStatus is the subset of a run's state the log path needs, resolved once
// and reused.
type logRunStatus struct {
	lifeCycleState string
	resultState    string
	stateMessage   string
	startTimeMs    int64
	endTimeMs      int64
	// latestAttempt is the highest attempt_number across the run's tasks.
	latestAttempt int
}

// A run is terminal when its lifecycle state is terminal, or a result state is
// set (result states only appear on terminal runs).
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

// resolveRunStatus fetches a run's state and projects it onto logRunStatus. An
// unknown run id surfaces as apierr.ErrResourceDoesNotExist.
func resolveRunStatus(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (logRunStatus, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return logRunStatus{}, err
	}
	return projectRunStatus(run), nil
}

// projectRunStatus extracts logRunStatus from a run. Split out so it can be
// tested without an API client.
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

// classifyLogError maps a Bricklens failure to one of:
//   - errBricklensFeatureDisabled: fall back to MLflow (gated off, endpoint
//     absent, or 404).
//   - the original error: a genuine not-found, surfaced as-is.
//   - nil: a transient failure the caller should retry.
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

// fromSeconds computes the `from` bound. With --minutes set it is now-N*60;
// otherwise the run's start second (0 before the run starts, which the endpoint
// reads as "everything stored").
func (req logRequest) fromSeconds(status logRunStatus, now time.Time) int64 {
	if req.windowMinutes > 0 {
		return now.Add(-time.Duration(req.windowMinutes) * time.Minute).Unix()
	}
	if status.startTimeMs > 0 {
		return status.startTimeMs / 1000
	}
	return 0
}

// toSeconds computes the `to` bound. A terminal run caps at its end second (ceil
// of the millisecond time, so the final partial second is kept); otherwise 0 lets
// the endpoint default to now.
func (req logRequest) toSeconds(status logRunStatus) int64 {
	if status.terminal() && status.endTimeMs > 0 {
		return (status.endTimeMs + 999) / 1000
	}
	return 0
}

// streamBricklensLogs fetches and prints a run's logs: a bounded tail for a
// completed run, or a poll-and-drain loop that follows an active run to
// completion. It returns whether the run finished with SUCCESS;
// errBricklensFeatureDisabled means the caller should fall back to MLflow.
func streamBricklensLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	// Build the API client once and reuse it for every page fetch in the loop.
	apiClient, err := client.New(w.Config)
	if err != nil {
		return false, fmt.Errorf("failed to create API client: %w", err)
	}
	st := &bricklensStreamer{
		ctx:       ctx,
		w:         w,
		apiClient: apiClient,
		out:       out,
		req:       req,
		status:    status,
		seen:      newSeenSet(seenRecordsCap),
	}
	return st.run()
}

// bricklensStreamer holds the poll-loop state: the from-second cursor, the
// highest emitted timestamp, and the dedup set.
type bricklensStreamer struct {
	ctx       context.Context
	w         *databricks.WorkspaceClient
	apiClient *client.DatabricksClient
	out       io.Writer
	req       logRequest
	status    logRunStatus

	fromSec      int64
	lastNano     int64
	firstLogSeen bool
	seen         *seenSet
	// previousState is the last display state reported to onStatusChange.
	previousState string
	// onFirstLog, when set, is called once just before the first log line is
	// emitted — used to stop the "waiting for run to start" spinner before any
	// log byte reaches stdout.
	onFirstLog func()
	// updateSpinner, when set, refreshes the waiting-spinner text each poll.
	updateSpinner func(string)
}

// reportStatusChange fires onStatusChange when the run's display state differs
// from the last reported one.
func (st *bricklensStreamer) reportStatusChange() {
	if st.req.onStatusChange == nil {
		return
	}
	current := st.status.displayState()
	if current == st.previousState {
		return
	}
	st.req.onStatusChange(current, st.previousState)
	st.previousState = current
}

func (st *bricklensStreamer) run() (bool, error) {
	now := time.Now()
	st.fromSec = st.req.fromSeconds(st.status, now)

	// A past retry's logs are immutable: render a one-shot tail rather than
	// following the still-active run, which would poll forever.
	if st.req.staticView {
		return st.drainStatic(st.req.toSeconds(st.status))
	}

	// Show a "waiting for run to start" spinner on stderr while the run has not
	// yet produced logs, so a provisioning run doesn't look hung. Suppressed in
	// --json mode and auto-degraded to nothing on a non-interactive terminal.
	// The first emitted log line stops it via onFirstLog (before any stdout write).
	if !st.req.jsonOutput {
		sp := cmdio.NewSpinner(st.ctx)
		defer sp.Close()
		st.onFirstLog = sp.Close
		st.updateSpinner = sp.Update
	}

	firstIteration := true
	for {
		if !firstIteration {
			status, err := resolveRunStatus(st.ctx, st.w, st.req.runID)
			if err != nil {
				if errors.Is(err, apierr.ErrResourceDoesNotExist) {
					return false, err
				}
				// A cancelled context (Ctrl-C) is not a transient blip: stop
				// promptly instead of retrying forever.
				if st.ctx.Err() != nil {
					return false, st.ctx.Err()
				}
				// A transient status blip should not abort a live stream.
				log.Debugf(st.ctx, "air logs: failed to refresh run status: %v", err)
				if err := sleepOrCancel(st.ctx, retryCheckInterval); err != nil {
					return false, err
				}
				continue
			}
			st.status = status
		}

		st.reportStatusChange()

		terminal := st.status.terminal()
		toSec := st.req.toSeconds(st.status)

		// While waiting on a still-active run with no logs yet, refresh the spinner.
		if !terminal && !st.firstLogSeen && st.updateSpinner != nil {
			st.updateSpinner(fmt.Sprintf("Waiting for run to start (node %d)...", st.req.node))
		}

		// A run already terminal on the first iteration renders as a tail (most
		// recent N lines). An active run streams everything with dedup, so a run
		// that terminates while we watch doesn't re-print the boundary second.
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
				// Stop the spinner before the no-logs line so frames don't smear.
				if st.onFirstLog != nil {
					st.onFirstLog()
				}
				st.emitNoLogs()
			}
			log.Infof(st.ctx, "air logs: run %d finished in state %s", st.req.runID, st.status.displayState())
			return st.status.succeeded(), nil
		}

		firstIteration = false
		if err := sleepOrCancel(st.ctx, retryCheckInterval); err != nil {
			return false, err
		}
	}
}

// sleepOrCancel waits for d, or returns early with the context error if the
// context is cancelled (e.g. Ctrl-C) so the poll loop exits promptly.
func sleepOrCancel(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// drainStatic renders a single tail pass without following the run. Success
// reflects the run's current result state (empty while active).
func (st *bricklensStreamer) drainStatic(toSec int64) (bool, error) {
	if err := st.drainTail(toSec); err != nil {
		return false, err
	}
	if !st.firstLogSeen {
		st.emitNoLogs()
	}
	return st.status.succeeded(), nil
}

// tailTarget is the number of lines a tail keeps. A negative tailLines means
// --lines was unset, so use the default cap; 0 or more is taken literally (an
// explicit --lines 0 prints nothing).
func (req logRequest) tailTarget() int {
	if req.tailLines < 0 {
		return defaultCompletedRunTailLines
	}
	return req.tailLines
}

// drainTail emits the most-recent `target` records oldest-first. Bricklens
// returns records newest-first, so it pages until it has `target`, keeps the
// newest `target`, and reverses to chronological order.
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

	// Keep the newest `target`, then reverse to print oldest -> newest.
	if len(collected) > target {
		collected = collected[:target]
	}
	for _, c := range slices.Backward(collected) {
		st.emit(c.Body)
	}
	return nil
}

// drainPages exhausts all pages from the current from-second in ascending order,
// deduping against the seen-set so a re-queried boundary second is not
// re-printed, then advances fromSec to the newest record's floor-second.
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
				// Skip a record older than the last emitted one to keep output
				// monotonic (out of order, or a re-queried boundary record).
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

// requestPage fetches one page, retrying transient failures up to
// maxTransientFailures before falling back to MLflow. A feature-gated response or
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
		resp, err := getBricklensLogs(st.ctx, st.apiClient, st.req.runID, q)
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
			log.Debugf(st.ctx, "air logs: bricklens failed %d times; falling back to mlflow", maxTransientFailures)
			return nil, errBricklensFeatureDisabled
		}
		log.Debugf(st.ctx, "air logs: bricklens transient failure (%d/%d): %v", transientFailures, maxTransientFailures, err)
		if err := sleepOrCancel(st.ctx, retryCheckInterval); err != nil {
			return nil, err
		}
	}
}

// emit writes one log line and latches firstLogSeen so an empty terminal run can
// report "no logs". The first line stops the waiting spinner before any byte
// reaches stdout.
func (st *bricklensStreamer) emit(body string) {
	if !st.firstLogSeen && st.onFirstLog != nil {
		st.onFirstLog()
	}
	st.firstLogSeen = true
	emitLogLine(st.out, st.req, body)
}

func (st *bricklensStreamer) emitNoLogs() {
	emitNoLogs(st.out, st.req, st.status)
}

// displayState is the result state, else the lifecycle state, else "UNKNOWN".
func (s logRunStatus) displayState() string {
	if s.resultState != "" {
		return s.resultState
	}
	if s.lifeCycleState != "" {
		return s.lifeCycleState
	}
	return "UNKNOWN"
}
