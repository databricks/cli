package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyLogError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error // errBricklensFeatureDisabled, the input error, or nil
	}{
		{
			name: "feature disabled falls back",
			err:  &apierr.APIError{ErrorCode: "FEATURE_DISABLED", StatusCode: http.StatusForbidden},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "endpoint not found falls back",
			err:  &apierr.APIError{ErrorCode: "ENDPOINT_NOT_FOUND", StatusCode: http.StatusNotFound},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "bare 404 falls back",
			err:  &apierr.APIError{ErrorCode: "SOMETHING", StatusCode: http.StatusNotFound},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "genuine resource-does-not-exist surfaces",
			err:  apierr.ErrResourceDoesNotExist,
			want: apierr.ErrResourceDoesNotExist,
		},
		{
			name: "transient 500 is retried",
			err:  &apierr.APIError{ErrorCode: "INTERNAL", StatusCode: http.StatusInternalServerError},
			want: nil,
		},
		{
			name: "plain error is retried",
			err:  errors.New("connection reset"),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLogError(tt.err)
			switch tt.want {
			case errBricklensFeatureDisabled:
				assert.ErrorIs(t, got, errBricklensFeatureDisabled)
			case nil:
				assert.NoError(t, got)
			default:
				assert.ErrorIs(t, got, tt.want)
			}
		})
	}
}

func TestProjectRunStatus(t *testing.T) {
	run := &jobs.Run{
		StartTime: 1000,
		EndTime:   2000,
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
			StateMessage:   "done",
		},
		Tasks: []jobs.RunTask{
			{AttemptNumber: 0},
			{AttemptNumber: 2},
			{AttemptNumber: 1},
		},
	}

	s := projectRunStatus(run)
	assert.Equal(t, "TERMINATED", s.lifeCycleState)
	assert.Equal(t, "SUCCESS", s.resultState)
	assert.Equal(t, "done", s.stateMessage)
	assert.Equal(t, int64(1000), s.startTimeMs)
	assert.Equal(t, int64(2000), s.endTimeMs)
	assert.Equal(t, 2, s.latestAttempt)
	assert.True(t, s.terminal())
	assert.True(t, s.succeeded())
	assert.Equal(t, "SUCCESS", s.displayState())
}

func TestLogRunStatusTerminal(t *testing.T) {
	tests := []struct {
		name         string
		lifeCycle    string
		resultState  string
		wantTerminal bool
	}{
		{"running", "RUNNING", "", false},
		{"pending", "PENDING", "", false},
		{"terminated lifecycle", "TERMINATED", "", true},
		{"internal error lifecycle", "INTERNAL_ERROR", "", true},
		{"failed result", "TERMINATING", "FAILED", true},
		{"canceled result", "RUNNING", "CANCELED", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := logRunStatus{lifeCycleState: tt.lifeCycle, resultState: tt.resultState}
			assert.Equal(t, tt.wantTerminal, s.terminal())
		})
	}
}

func TestLogRequestFromSeconds(t *testing.T) {
	now := time.Unix(10_000, 0)

	// --minutes narrows the window to now - N*60.
	req := logRequest{windowMinutes: 5}
	assert.Equal(t, int64(10_000-300), req.fromSeconds(logRunStatus{startTimeMs: 1_000_000}, now))

	// No window: from the run's start second.
	req = logRequest{}
	assert.Equal(t, int64(1000), req.fromSeconds(logRunStatus{startTimeMs: 1_000_000}, now))

	// No window, run not started: everything stored (0).
	assert.Equal(t, int64(0), req.fromSeconds(logRunStatus{}, now))
}

func TestLogRequestToSeconds(t *testing.T) {
	req := logRequest{}

	// Active run: 0 lets the endpoint default to now.
	assert.Equal(t, int64(0), req.toSeconds(logRunStatus{lifeCycleState: "RUNNING"}))

	// Terminal run: ceil of the end millisecond so the final partial second is kept.
	terminal := logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS", endTimeMs: 2001}
	assert.Equal(t, int64(3), req.toSeconds(terminal))
}

func TestLogRequestTailTarget(t *testing.T) {
	assert.Equal(t, defaultCompletedRunTailLines, logRequest{}.tailTarget())
	assert.Equal(t, 42, logRequest{tailLines: 42}.tailTarget())
}

func TestDrainPagesDedupAndOrdering(t *testing.T) {
	// Two pages: page 1 has two ascending records; page 2 repeats the last record
	// of page 1 (boundary re-query — must dedup) and includes an older record
	// (out of order — must skip), then a genuinely newer one.
	var page int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_token") == "" {
			page = 1
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 1000, "body": "a", "node_index": 0},
				{"time_unix_nano": 2000, "body": "b", "node_index": 0}
			], "next_page_token": "p2"}`))
			return
		}
		page = 2
		_, _ = w.Write([]byte(`{"log_records": [
			{"time_unix_nano": 2000, "body": "b", "node_index": 0},
			{"time_unix_nano": 1500, "body": "stale", "node_index": 0},
			{"time_unix_nano": 3000, "body": "c", "node_index": 0}
		]}`))
	}))
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	st := &bricklensStreamer{
		ctx:  t.Context(),
		w:    newTestWorkspaceClient(t, srv.URL),
		out:  &buf,
		req:  logRequest{runID: 1, node: 0, attempt: -1},
		seen: newSeenSet(seenRecordsCap),
	}
	require.NoError(t, st.drainPages(0))
	require.Equal(t, 2, page)

	// "b" prints once (deduped), "stale" is skipped (older than last emitted), and
	// fromSec advances to the newest record's floor-second (3000ns -> 0s here).
	assert.Equal(t, "a\nb\nc\n", buf.String())
	assert.Equal(t, int64(3000), st.lastNano)
}

func TestDisplayState(t *testing.T) {
	assert.Equal(t, "SUCCESS", logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS"}.displayState())
	assert.Equal(t, "RUNNING", logRunStatus{lifeCycleState: "RUNNING"}.displayState())
	assert.Equal(t, "UNKNOWN", logRunStatus{}.displayState())
}

func TestEmitLogLineJSON(t *testing.T) {
	var buf bytes.Buffer
	emitLogLine(&buf, logRequest{node: 2, jsonOutput: true}, "hello")

	var ev logEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &ev))
	assert.Equal(t, "LOG", ev.Type)
	assert.Equal(t, 2, ev.Node)
	assert.Equal(t, "hello", ev.Line)
	assert.NotEmpty(t, ev.TS)
}

func TestEmitLogLineText(t *testing.T) {
	var buf bytes.Buffer
	emitLogLine(&buf, logRequest{node: 0}, "hello")
	assert.Equal(t, "hello\n", buf.String())
}

func TestEmitNoLogs(t *testing.T) {
	status := logRunStatus{lifeCycleState: "TERMINATED", resultState: "FAILED", stateMessage: "boom"}

	var text bytes.Buffer
	emitNoLogs(&text, logRequest{runID: 7}, status)
	assert.Equal(t, "No logs available for run 7. Run terminated in state FAILED: boom\n", text.String())

	var jsonBuf bytes.Buffer
	emitNoLogs(&jsonBuf, logRequest{runID: 7, node: 1, jsonOutput: true}, status)
	var ev logEvent
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &ev))
	assert.Equal(t, "ERROR", ev.Type)
	assert.Equal(t, 1, ev.Node)
	assert.Contains(t, ev.Line, "No logs available for run 7")
	assert.Contains(t, ev.Line, "boom")
}

func TestRequestPageRetriesThenFallsBack(t *testing.T) {
	// Shrink the retry wait so the transient-failure loop runs fast.
	orig := retryCheckInterval
	retryCheckInterval = time.Millisecond
	t.Cleanup(func() { retryCheckInterval = orig })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/logs") {
			// Ignore SDK host/config probes so `calls` counts only log requests.
			_, _ = w.Write([]byte(`{}`))
			return
		}
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code": "INTERNAL_ERROR", "message": "transient"}`))
	}))
	t.Cleanup(srv.Close)

	st := &bricklensStreamer{
		ctx:  t.Context(),
		w:    newTestWorkspaceClient(t, srv.URL),
		req:  logRequest{runID: 1, node: 0, attempt: -1},
		seen: newSeenSet(seenRecordsCap),
	}
	_, err := st.requestPage("", 0, 0, true)
	// Persistent transient failures fall back to MLflow after the retry budget.
	require.ErrorIs(t, err, errBricklensFeatureDisabled)
	assert.Equal(t, maxTransientFailures, calls)
}

func TestRequestPageRetriesThenSucceeds(t *testing.T) {
	orig := retryCheckInterval
	retryCheckInterval = time.Millisecond
	t.Cleanup(func() { retryCheckInterval = orig })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/logs") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error_code": "INTERNAL_ERROR", "message": "transient"}`))
			return
		}
		_, _ = w.Write([]byte(`{"log_records": [{"time_unix_nano": 1, "body": "ok", "node_index": 0}]}`))
	}))
	t.Cleanup(srv.Close)

	st := &bricklensStreamer{
		ctx:  t.Context(),
		w:    newTestWorkspaceClient(t, srv.URL),
		req:  logRequest{runID: 1, node: 0, attempt: -1},
		seen: newSeenSet(seenRecordsCap),
	}
	resp, err := st.requestPage("", 0, 0, true)
	require.NoError(t, err)
	require.Len(t, resp.LogRecords, 1)
	assert.Equal(t, "ok", resp.LogRecords[0].Body)
	assert.Equal(t, 3, calls)
}

func TestSeenSetEviction(t *testing.T) {
	s := newSeenSet(2)
	s.add(1, "a")
	s.add(2, "b")
	assert.True(t, s.has(1, "a"))
	assert.True(t, s.has(2, "b"))

	// Adding a third evicts the oldest-inserted (1,"a").
	s.add(3, "c")
	assert.False(t, s.has(1, "a"))
	assert.True(t, s.has(2, "b"))
	assert.True(t, s.has(3, "c"))

	// Same (nano, body) shares one entry; distinct body under the same nano does not.
	s.add(3, "c")
	assert.True(t, s.has(3, "c"))
	assert.False(t, s.has(3, "d"))
}
