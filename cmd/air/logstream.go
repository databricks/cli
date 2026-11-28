package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
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
	// --tail nor --minutes is set.
	defaultCompletedRunTailLines = 10000
	// seenRecordsCap bounds the dedup set, evicting oldest-inserted entries first.
	seenRecordsCap = 100000
	// statusMessageRefreshEveryNPolls throttles the status_message fetch so the
	// waiting spinner doesn't issue a get-output on every poll tick.
	statusMessageRefreshEveryNPolls = 5
	// bricklensEmptyMLflowProbeEveryNPolls throttles the active-run MLflow
	// existence probe while Bricklens has returned no records.
	bricklensEmptyMLflowProbeEveryNPolls = 10
	// Event time and ingestion order can differ, so live polls overlap.
	bricklensLogLookback = 30 * time.Second
	// A terminal job can have records that are not searchable yet.
	bricklensTerminalEmptyPolls = 3
	// Bricklens keeps indexing a run's records for a short while after it ends, so
	// a tail read right after the end can miss lines from the middle of the log.
	bricklensIndexSettleWindow = 2 * time.Minute
)

// statusMessageType tags a client-facing message packed into
// ai_runtime_task_output.status_message as "<TYPE>:<payload>"; only STATUS-typed
// messages are surfaced.
const statusMessageType = "STATUS"

// waitingForComputeStatus is the normalized form of AIR's capacity status.
const waitingForComputeStatus = "Waiting for accelerator compute capacity to become available..."

// waitingForWorkloadStatus avoids inferring why a Jobs run or task appears
// pending when AIR does not provide a current status message.
const waitingForWorkloadStatus = "Waiting for the latest workload status..."

// normalizeStatusMessage returns the payload of a "STATUS:<payload>" message,
// normalized for display (trailing "." stripped, "..." suffix added), or "" for
// any other type or an empty/absent message.
func normalizeStatusMessage(raw string) string {
	messageType, payload, ok := strings.Cut(raw, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(messageType), statusMessageType) {
		return ""
	}
	payload = strings.TrimSpace(payload)
	payload = strings.TrimRight(payload, ".")
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return ""
	}
	if strings.EqualFold(payload, "Waiting for GPU compute capacity to become available") {
		return waitingForComputeStatus
	}
	return payload + "..."
}

// retryCheckInterval is the wait between status/log polls. A var so tests can
// shrink it.
var retryCheckInterval = 3 * time.Second

// bricklensPollInterval is independent of slower retry and discovery backoffs.
var bricklensPollInterval = time.Second

// bricklensSummarySettleFloor is the least time between the execution summary
// ending the stream and the terminal poll for the summary to end the stream
// early. It keeps a margin above the longest indexing delay measured for a
// record (a few seconds) even when the run turns terminal right after the
// summary.
var bricklensSummarySettleFloor = 5 * time.Second

// errBricklensFeatureDisabled signals the caller to fall back to MLflow: Bricklens
// is gated off (FEATURE_DISABLED), not deployed (ENDPOINT_NOT_FOUND / 404),
// persistently failing, or served every request successfully but never returned a
// record for a run whose logs may still be in MLflow. The flag is evaluated
// server-side.
var errBricklensFeatureDisabled = errors.New("bricklens logs unavailable; falling back to mlflow")

// logRequest describes what to fetch, shared by both backends so they honor the
// same flags. windowMinutes and tailLines are mutually exclusive.
type logRequest struct {
	runID int64
	// runURL preserves the URL already shown by air run --watch.
	runURL string
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
	// --tail was unset (use the default cap); 0 prints no log lines.
	tailLines int
	// downloadTo, when set, writes logs to that directory instead of stdout.
	downloadTo string
	// staticView renders snapshots and immutable past retries without following an active run.
	staticView bool
	// printOnce prints an active run's available logs and exits instead of
	// following it (air logs without --follow).
	printOnce bool
	// boundInitialLogs limits existing output before following an active run.
	boundInitialLogs bool
	jsonOutput       bool
	maxRetries       *int
	// onStatusChange, when set, is called on each lifecycle transition while
	// following the run (current, previous display states). Used by
	// `air run --watch -o json` to emit STATUS events.
	onStatusChange func(current, previous string)
	retryTracker   *retryTracker
}

// logRunStatus is the subset of a run's state the log path needs, resolved once
// and reused.
type logRunStatus struct {
	lifeCycleState           string
	latestTaskLifeCycleState string
	resultState              string
	stateMessage             string
	startTimeMs              int64
	endTimeMs                int64
	// latestAttempt is the highest attempt_number across the run's tasks.
	latestAttempt int
	// attemptEndTimeMs maps each finished attempt_number to its task's end time.
	attemptEndTimeMs map[int]int64
}

// retryTracker reports each retry once across either Jobs transition:
//
//  1. Attempt N enters WAITING_FOR_RETRY: report retry N+1.
//  2. Attempt N+1 appears: report retry N+1 only if transition 1 was missed.
type retryTracker struct {
	lastReportedRetry           int
	lastObservedAttempt         int
	lastObservedWaitingForRetry bool
}

func newRetryTracker(status logRunStatus) *retryTracker {
	return &retryTracker{
		lastReportedRetry:           status.latestAttempt,
		lastObservedAttempt:         status.latestAttempt,
		lastObservedWaitingForRetry: status.latestTaskLifeCycleState == string(jobs.RunLifeCycleStateWaitingForRetry),
	}
}

func (t *retryTracker) observe(status logRunStatus) (int, bool) {
	waitingForRetry := status.latestTaskLifeCycleState == string(jobs.RunLifeCycleStateWaitingForRetry)
	retry := 0
	if status.latestAttempt > t.lastObservedAttempt {
		t.lastObservedAttempt = status.latestAttempt
		t.lastObservedWaitingForRetry = waitingForRetry
		retry = status.latestAttempt
	} else if status.latestAttempt == t.lastObservedAttempt {
		if waitingForRetry && !t.lastObservedWaitingForRetry {
			retry = status.latestAttempt + 1
		}
		t.lastObservedWaitingForRetry = waitingForRetry
	}
	if retry <= t.lastReportedRetry || retry <= 0 {
		return 0, false
	}
	t.lastReportedRetry = retry
	return retry, true
}

func (req logRequest) reportRetry(ctx context.Context, out io.Writer, status logRunStatus, before func()) {
	if req.retryTracker == nil {
		return
	}
	retry, ok := req.retryTracker.observe(status)
	if !ok {
		return
	}
	if before != nil {
		before()
	}
	if req.jsonOutput {
		printRetryEvent(out, retry, req.maxRetries)
	} else if req.maxRetries != nil {
		cmdio.LogString(ctx, fmt.Sprintf("\nRetrying (%d of %d)...\n", retry, *req.maxRetries))
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("\nRetrying (%d)...\n", retry))
	}
}

// A run is terminal when its lifecycle state is terminal, or a result state is
// set (result states only appear on terminal runs).
var (
	terminalLifeCycleStates = map[string]bool{"TERMINATED": true, "SKIPPED": true, "INTERNAL_ERROR": true}
	terminalResultStates    = map[string]bool{"SUCCESS": true, "FAILED": true, "TIMEDOUT": true, "CANCELED": true}
)

func (s logRunStatus) terminal() bool {
	return terminalLifeCycleStates[s.lifeCycleState] || terminalResultStates[s.resultState]
}

// attemptEndedWithin reports whether the given attempt (-1 for the latest) of a
// terminal run ended less than d before now. A past attempt uses its own task's
// end time: it may have ended long before the run did.
func (s logRunStatus) attemptEndedWithin(attempt int, now time.Time, d time.Duration) bool {
	if !s.terminal() {
		return false
	}
	endMs := s.endTimeMs
	if attempt >= 0 && attempt != s.latestAttempt {
		endMs = s.attemptEndTimeMs[attempt]
	}
	return endMs > 0 && now.Sub(time.UnixMilli(endMs)) < d
}

func (s logRunStatus) succeeded() bool {
	return s.resultState == "SUCCESS"
}

func (s logRunStatus) appearsPending() bool {
	if s.resultState != "" {
		return false
	}
	if s.lifeCycleState == string(jobs.RunLifeCycleStatePending) {
		return true
	}
	if s.lifeCycleState != string(jobs.RunLifeCycleStateRunning) {
		return false
	}
	switch jobs.RunLifeCycleState(s.latestTaskLifeCycleState) {
	case jobs.RunLifeCycleStatePending, jobs.RunLifeCycleStateQueued,
		jobs.RunLifeCycleStateWaitingForRetry, jobs.RunLifeCycleStateBlocked:
		return true
	default:
		return false
	}
}

// downloadOutcome is the exit status for a one-shot fetch, which unlike streaming
// can run against an active run. An active run has no result state yet, and
// not-yet-finished is not a failure, so only a terminal run decides the exit code.
func (s logRunStatus) downloadOutcome() bool {
	return !s.terminal() || s.succeeded()
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
		task := &run.Tasks[i]
		if task.EndTime > 0 {
			if s.attemptEndTimeMs == nil {
				s.attemptEndTimeMs = map[int]int64{}
			}
			s.attemptEndTimeMs[task.AttemptNumber] = max(s.attemptEndTimeMs[task.AttemptNumber], task.EndTime)
		}
		if i > 0 && task.AttemptNumber <= s.latestAttempt {
			continue
		}
		s.latestAttempt = task.AttemptNumber
		s.latestTaskLifeCycleState = ""
		if task.State != nil {
			s.latestTaskLifeCycleState = string(task.State.LifeCycleState)
		}
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
	success, _, err := streamBricklensLogsWithLatestStatus(ctx, w, out, req, status)
	return success, err
}

// streamBricklensLogsWithLatestStatus is streamBricklensLogs that also returns
// the most recent run status the stream observed.
func streamBricklensLogsWithLatestStatus(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, logRunStatus, error) {
	// Build the API client once and reuse it for every page fetch in the loop.
	apiClient, err := client.New(w.Config)
	if err != nil {
		return false, status, fmt.Errorf("failed to create API client: %w", err)
	}
	st := &bricklensStreamer{
		ctx:       ctx,
		w:         w,
		apiClient: apiClient,
		headers:   auth.WorkspaceIDHeaders(w.Config),
		out:       out,
		req:       req,
		status:    status,
		seen:      newSeenSet(seenRecordsCap),
	}
	success, err := st.run()
	return success, st.status, err
}

// bricklensStreamer holds the poll-loop state: the from-second cursor, the
// highest emitted timestamp, and the dedup set.
type bricklensStreamer struct {
	ctx       context.Context
	w         *databricks.WorkspaceClient
	apiClient *client.DatabricksClient
	headers   map[string]string
	out       io.Writer
	req       logRequest
	status    logRunStatus

	fromSec        int64
	streamStartSec int64
	lastNano       int64
	firstLogSeen   bool
	seen           *seenSet
	// previousState is the last display state reported to onStatusChange.
	previousState string
	// onFirstLog, when set, is called once just before the first log line is
	// emitted — used to stop the "waiting for run to start" spinner before any
	// log byte reaches stdout.
	onFirstLog func()
	// updateSpinner, when set, refreshes the waiting-spinner text each poll.
	updateSpinner func(string)
	// summary tracks whether the stream ends with the entry script's execution
	// summary; summaryAttempt is the latest attempt and summarySeenAt the time
	// when that summary ended it.
	summary        summaryTracker
	summaryAttempt int
	summarySeenAt  time.Time
	// quietPollsAfterSummary counts consecutive polls that emitted nothing
	// since the summary ended the stream.
	quietPollsAfterSummary int
}

// tailEvidence names what showed a terminal run's log stream is complete.
type tailEvidence int

const (
	tailIncomplete tailEvidence = iota
	// The latest attempt's execution summary ended the stream at least
	// bricklensSummarySettleFloor before the terminal poll, and at least
	// bricklensTerminalEmptyPolls polls since then returned nothing. The
	// summary is the entry script's last output, so those polls already gave
	// late records the time the terminal empty polls would.
	tailSettledSummary
	// bricklensTerminalEmptyPolls consecutive terminal polls returned nothing.
	tailTerminalEmptyPolls
)

// tailComplete decides whether a terminal run's stream can stop, given the
// consecutive empty terminal polls so far and when the current terminal poll
// started. Evidence is checked strongest first; without any, keep polling.
func (st *bricklensStreamer) tailComplete(terminalEmptyPolls int, terminalPollStarted time.Time) tailEvidence {
	switch {
	case st.summary.ended &&
		st.summaryAttempt == st.status.latestAttempt &&
		st.quietPollsAfterSummary >= bricklensTerminalEmptyPolls &&
		terminalPollStarted.Sub(st.summarySeenAt) >= bricklensSummarySettleFloor:
		return tailSettledSummary
	case terminalEmptyPolls >= bricklensTerminalEmptyPolls:
		return tailTerminalEmptyPolls
	default:
		return tailIncomplete
	}
}

// waitingSpinnerText returns the waiting-spinner text: the server-set STATUS
// message if present, else a neutral message for a run or task that appears
// pending, else the default "waiting for run to start".
func (st *bricklensStreamer) waitingSpinnerText() string {
	if msg := st.serverStatusMessage(); msg != "" {
		return msg
	}
	if st.status.appearsPending() {
		return waitingForWorkloadStatus
	}
	return fmt.Sprintf("Waiting for run to start (node %d)...", st.req.node)
}

// serverStatusMessage returns the run's server-set STATUS message (normalized for
// display), or "" if unavailable. The message lives on the latest task run's
// output, re-resolved each call so a retry's new task run is picked up.
// Best-effort: any fetch failure logs at debug and returns "".
func (st *bricklensStreamer) serverStatusMessage() string {
	run, err := st.w.Jobs.GetRun(st.ctx, jobs.GetRunRequest{RunId: st.req.runID})
	if err != nil || len(run.Tasks) == 0 {
		return ""
	}
	taskRunID := run.Tasks[len(run.Tasks)-1].RunId
	out, err := st.w.Jobs.GetRunOutputByRunId(st.ctx, taskRunID)
	if err != nil {
		log.Debugf(st.ctx, "air logs: status_message fetch failed for run %d: %v", st.req.runID, err)
		return ""
	}
	if out.AiRuntimeTaskOutput == nil {
		return ""
	}
	return normalizeStatusMessage(out.AiRuntimeTaskOutput.StatusMessage)
}

// logWaitingStatus prints a waiting status as a plain stderr line, for terminals
// that can't show the spinner.
func (st *bricklensStreamer) logWaitingStatus(msg string) {
	cmdio.LogProgress(st.ctx, msg)
}

// printWaitingEvent reports a waiting status as a WAITING JSONL event in place of
// the spinner, which -o json never shows.
func (st *bricklensStreamer) printWaitingEvent(msg string) {
	printWaitingEvent(st.out, msg)
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
	st.streamStartSec = st.fromSec

	if st.req.staticView {
		return st.drainStatic(st.req.toSeconds(st.status))
	}

	// Report the waiting status while the run has not yet produced logs, so a
	// provisioning run doesn't look hung: a spinner on an interactive stderr, one
	// line per status change on a non-interactive stderr, and WAITING events in
	// --json mode. The first emitted log line stops the spinner via onFirstLog
	// (before any stdout write).
	if st.req.jsonOutput {
		st.updateSpinner = st.printWaitingEvent
	} else {
		sp := cmdio.NewSpinner(st.ctx)
		defer sp.Close()
		st.onFirstLog = sp.Close
		st.updateSpinner = sp.Update
		// A non-interactive stderr (agents, CI, pipes) gets no spinner, which made
		// a provisioning run look hung. Print each new waiting status as one line
		// instead; the caller only calls updateSpinner when the text changes.
		if cmdio.GetInteractiveMode(st.ctx) == cmdio.InteractiveModeNone {
			st.updateSpinner = st.logWaitingStatus
		}
	}

	firstIteration := true
	// Throttled refresh of the waiting-spinner text: statusRefreshCounter gates the
	// server status_message fetch to every Nth poll, and lastSpinnerText avoids
	// redundant spinner updates.
	statusRefreshCounter := 0
	emptyStreamPolls := 0
	terminalEmptyPolls := 0
	lastSpinnerText := ""
	for {
		pollStarted := time.Now()
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
		st.req.reportRetry(st.ctx, st.out, st.status, st.onFirstLog)

		terminal := st.status.terminal()
		toSec := st.req.toSeconds(st.status)

		// Remember an active attachment's initial tail so overlapping polls do not reprint records.
		var emitted int
		var err error
		if firstIteration && (terminal || st.req.boundInitialLogs) {
			err = st.drainTail(toSec, !terminal)
			if err == nil && !terminal {
				// Baseline omitted history without suppressing newer records.
				_, err = st.drainPages(toSec, true)
			}
		} else {
			emitted, err = st.drainPages(toSec, false)
		}
		if err != nil {
			return false, err
		}

		if !firstIteration {
			if emitted == 0 {
				st.quietPollsAfterSummary++
			} else {
				st.quietPollsAfterSummary = 0
			}
		}

		if terminal {
			if !firstIteration {
				if emitted == 0 {
					terminalEmptyPolls++
				} else {
					terminalEmptyPolls = 0
				}
				if st.tailComplete(terminalEmptyPolls, pollStarted) == tailIncomplete {
					if err := waitForNextBricklensPoll(st.ctx, pollStarted); err != nil {
						return false, err
					}
					firstIteration = false
					continue
				}
			}
			if !st.firstLogSeen {
				// A successful but empty Bricklens stream isn't proof the run has no
				// logs; they may be in MLflow (as --download-to reads). Fall back
				// there, which owns the real no-logs report and the same exit code.
				return false, errBricklensFeatureDisabled
			}
			log.Infof(st.ctx, "air logs: run %d finished in state %s", st.req.runID, st.status.displayState())
			return st.status.succeeded(), nil
		}
		if !st.firstLogSeen {
			// The fetch above found no logs for this still-active run, so refresh the
			// waiting status with the server-set message (throttled): a waiting run
			// shows any AIR-provided detail rather than a generic message. Doing this
			// after the fetch means attaching to a run that already has logs never
			// reports it as waiting.
			if st.updateSpinner != nil {
				if statusRefreshCounter%statusMessageRefreshEveryNPolls == 0 {
					if desired := st.waitingSpinnerText(); desired != lastSpinnerText {
						st.updateSpinner(desired)
						lastSpinnerText = desired
					}
				}
				statusRefreshCounter++
			}
			emptyStreamPolls++
			if emptyStreamPolls%bricklensEmptyMLflowProbeEveryNPolls == 0 && mlflowLogsExist(st.ctx, st.w, st.req) {
				log.Debugf(st.ctx, "air logs: MLflow artifacts exist for run %d; falling back to MLflow log stream", st.req.runID)
				return false, errBricklensFeatureDisabled
			}
		}

		firstIteration = false
		if err := waitForNextBricklensPoll(st.ctx, pollStarted); err != nil {
			return false, err
		}
	}
}

func waitForNextBricklensPoll(ctx context.Context, pollStarted time.Time) error {
	return sleepOrCancel(ctx, max(time.Duration(0), bricklensPollInterval-time.Since(pollStarted)))
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
	if err := st.drainTail(toSec, false); err != nil {
		return false, err
	}
	if !st.firstLogSeen {
		// An empty Bricklens tail doesn't mean the attempt has no logs; fall back to
		// MLflow, which holds the immutable per-attempt artifacts. See the terminal
		// branch in run.
		return false, errBricklensFeatureDisabled
	}
	return st.status.downloadOutcome(), nil
}

// tailTarget is the number of lines a tail keeps. A negative tailLines means
// --tail was unset, so use the default cap; 0 or more is taken literally (an
// explicit --tail 0 prints no log lines).
func (req logRequest) tailTarget() int {
	if req.tailLines < 0 {
		return defaultCompletedRunTailLines
	}
	return req.tailLines
}

// drainTail emits the most-recent `target` records oldest-first. Bricklens
// returns records newest-first, so it pages until it has `target`, keeps the
// newest `target`, and reverses to chronological order.
func (st *bricklensStreamer) drainTail(toSec int64, remember bool) error {
	collected, err := st.fetchTail(toSec)
	if err != nil {
		return err
	}
	if st.status.attemptEndedWithin(st.req.attempt, time.Now(), bricklensIndexSettleWindow) {
		collected, err = st.settleTail(toSec, collected)
		if err != nil {
			return err
		}
	}

	// collected is newest first; print oldest -> newest.
	for _, c := range slices.Backward(collected) {
		st.emit(c.Body)
		if remember {
			st.seen.add(c)
			st.lastNano = max(st.lastNano, c.nano())
		}
	}
	if remember && st.lastNano != 0 {
		st.fromSec = max(st.streamStartSec, st.lastNano/1_000_000_000-int64(bricklensLogLookback/time.Second))
	}
	return nil
}

// settleTail re-reads the tail of a run that just ended until two reads return
// the same records, so lines Bricklens indexes late are not dropped. Comparing
// identities rather than counts matters once the tail is full: a late record
// then shifts the window without changing its length.
func (st *bricklensStreamer) settleTail(toSec int64, collected []logRecord) ([]logRecord, error) {
	for range bricklensTerminalEmptyPolls {
		if err := sleepOrCancel(st.ctx, bricklensPollInterval); err != nil {
			return nil, err
		}
		next, err := st.fetchTail(toSec)
		if err != nil {
			return nil, err
		}
		if slices.EqualFunc(next, collected, func(a, b logRecord) bool { return seenRecordKey(a) == seenRecordKey(b) }) {
			return next, nil
		}
		collected = next
	}
	return collected, nil
}

// fetchTail returns up to the tail target's newest records, newest first.
func (st *bricklensStreamer) fetchTail(toSec int64) ([]logRecord, error) {
	target := st.req.tailTarget()
	if target <= 0 {
		return nil, nil
	}

	var collected []logRecord
	var pageToken string
	for len(collected) < target {
		resp, err := st.requestPage(pageToken, toSec, target, false)
		if err != nil {
			return nil, err
		}
		collected = append(collected, resp.LogRecords...)
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	if len(collected) > target {
		collected = collected[:target]
	}
	return collected, nil
}

// drainPages exhausts all pages from the current from-second in ascending order.
// Live polls retain a bounded overlap so late records remain visible. The
// initial baseline suppresses omitted history through the tail's watermark.
func (st *bricklensStreamer) drainPages(toSec int64, suppressInitialHistory bool) (int, error) {
	emitted := 0
	initialTailNano := st.lastNano
	var maximumEvictedNano int64
	var pageToken string
	for {
		resp, err := st.requestPage(pageToken, toSec, 0, true)
		if err != nil {
			return 0, err
		}

		for _, rec := range resp.LogRecords {
			nano := rec.nano()
			if st.seen.has(rec) {
				continue
			}
			if !suppressInitialHistory || (nano != 0 && nano > initialTailNano) {
				st.emit(rec.Body)
				emitted++
			}
			maximumEvictedNano = max(maximumEvictedNano, st.seen.add(rec))
			if nano != 0 {
				st.lastNano = max(st.lastNano, nano)
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	if st.lastNano != 0 {
		st.fromSec = max(st.streamStartSec, st.lastNano/1_000_000_000-int64(bricklensLogLookback/time.Second))
		st.seen.removeBefore(st.fromSec * 1_000_000_000)
		if maximumEvictedNano != 0 {
			st.fromSec = max(st.fromSec, maximumEvictedNano/1_000_000_000+1)
		}
	}
	return emitted, nil
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
		resp, err := getBricklensLogs(st.ctx, st.apiClient, st.headers, st.req.runID, q)
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
	st.summary.observe(body)
	if st.summary.ended {
		st.summaryAttempt = st.status.latestAttempt
		st.summarySeenAt = time.Now()
	}
	emitLogLine(st.out, st.req, body)
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
