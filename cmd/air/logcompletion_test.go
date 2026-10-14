package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var successSummary = []string{
	"================================",
	"Execution Summary",
	"================================",
	"Total time: 32s",
	"User code execution time: 32s",
	"Completed at Fri Oct  9 09:46:18 PM UTC 2026",
	"================================",
}

var failureSummary = []string{
	"================================",
	"Execution Summary",
	"================================",
	"Total time: 32s",
	"User code execution time: 32s",
	"ERROR: Script failed with exit code 1 after 32s",
	"================================",
}

func TestCompletionMatcher(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  bool
	}{
		{name: "success summary", lines: successSummary, want: true},
		{
			name:  "success summary without user code time",
			lines: slices.Delete(slices.Clone(successSummary), 4, 5),
			want:  true,
		},
		{name: "failure summary", lines: failureSummary, want: false},
		{
			name:  "trailing whitespace",
			lines: []string{"Execution Summary\r", "====\r", "Total time: 1s", "Completed at now\r"},
			want:  true,
		},
		{
			name:  "interrupted block",
			lines: []string{"Execution Summary", "====", "step 5", "Total time: 1s", "Completed at now"},
			want:  false,
		},
		{name: "completed line alone", lines: []string{"Completed at epoch 3"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m completionMatcher
			for _, line := range tt.lines {
				m.observe(line)
			}
			assert.Equal(t, tt.want, m.completed)
		})
	}
}

// logsBody renders a log endpoint response for the streamed node. Records are
// timestamped one second apart from 1700000000s and keyed by position and
// content, so a poll that re-serves earlier lines dedups them while new lines,
// including the summary's repeated rules, stay distinct.
func logsBody(t *testing.T, lines []string) string {
	t.Helper()
	type record struct {
		RecordID     string `json:"record_id"`
		TimeUnixNano int64  `json:"time_unix_nano"`
		Body         string `json:"body"`
	}
	records := make([]record, len(lines))
	for i, line := range lines {
		records[i] = record{
			RecordID:     strconv.Itoa(i) + ":" + line,
			TimeUnixNano: (1_700_000_000 + int64(i)) * 1_000_000_000,
			Body:         line,
		}
	}
	b, err := json.Marshal(map[string]any{"log_records": records})
	assert.NoError(t, err)
	return string(b)
}

const (
	runningRunBody   = `{"run_id":1,"start_time":1700000000000,"state":{"life_cycle_state":"RUNNING"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	succeededRunBody = `{"run_id":1,"start_time":1700000000000,"end_time":1700000100000,"state":{"life_cycle_state":"TERMINATED","result_state":"SUCCESS"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	failedRunBody    = `{"run_id":1,"start_time":1700000000000,"end_time":1700000100000,"state":{"life_cycle_state":"TERMINATED","result_state":"FAILED"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	retryingRunBody  = `{"run_id":1,"start_time":1700000000000,"state":{"life_cycle_state":"RUNNING"},"tasks":[{"run_id":2,"attempt_number":0},{"run_id":3,"attempt_number":1}]}`
)

// completionServer counts the requests the streamer made to the fake server.
type completionServer struct {
	runGets     int
	logRequests int
}

// newCompletionServer serves an active run. runBodies are returned by successive
// runs/get calls (the last repeats); lines returns node 0's log lines for the
// n-th log request (1-based).
func newCompletionServer(t *testing.T, runBodies []string, lines func(n int) []string) (*completionServer, *httptest.Server) {
	t.Helper()
	original := bricklensPollInterval
	bricklensPollInterval = time.Millisecond
	t.Cleanup(func() { bricklensPollInterval = original })

	cs := &completionServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/logs"):
			assert.Equal(t, "0", r.URL.Query().Get("filter.node_index"))
			cs.logRequests++
			_, _ = w.Write([]byte(logsBody(t, lines(cs.logRequests))))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(runBodies[min(cs.runGets, len(runBodies)-1)]))
			cs.runGets++
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return cs, srv
}

func streamForCompletion(t *testing.T, srvURL string, exitOnCompletion bool) (bool, string, error) {
	t.Helper()
	// Every scenario ends within a few polls; fail rather than hang if one does not.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var buf bytes.Buffer
	ok, err := streamBricklensLogs(ctx, newTestWorkspaceClient(t, srvURL), &buf,
		logRequest{runID: 1, attempt: -1, tailLines: -1, jsonOutput: true, exitOnCompletion: exitOnCompletion},
		logRunStatus{lifeCycleState: "RUNNING", startTimeMs: 1_700_000_000_000})
	return ok, buf.String(), err
}

func TestStreamBricklensExitsOnSuccessSummary(t *testing.T) {
	cs, srv := newCompletionServer(t, []string{runningRunBody}, func(int) []string {
		return append([]string{"training"}, successSummary...)
	})

	ok, out, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	assert.True(t, ok)
	// Exited on the first poll, without Jobs ever reporting the run terminal.
	assert.Equal(t, 0, cs.runGets)
	assert.Equal(t, 1, cs.logRequests)
	assert.Contains(t, out, `"line":"training"`)
	assert.Contains(t, out, `"line":"Completed at `)
}

func TestStreamBricklensMatchesSummarySplitAcrossPolls(t *testing.T) {
	cs, srv := newCompletionServer(t, []string{runningRunBody}, func(n int) []string {
		lines := append([]string{"training"}, successSummary...)
		if n == 1 {
			return lines[:4]
		}
		return lines
	})

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 2, cs.logRequests)
	assert.Equal(t, 1, cs.runGets)
}

func TestStreamBricklensIgnoresFailureSummary(t *testing.T) {
	cs, srv := newCompletionServer(t, []string{runningRunBody, failedRunBody}, func(int) []string {
		return failureSummary
	})

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	// Waited for Jobs: a failed attempt may still be retried.
	assert.False(t, ok)
	assert.GreaterOrEqual(t, cs.runGets, 2)
}

func TestStreamBricklensExitsOnRetrySuccess(t *testing.T) {
	// Attempt 0 fails, so the stream keeps following the run; the retry then
	// succeeds and the stream exits without waiting for Jobs.
	cs, srv := newCompletionServer(t, []string{retryingRunBody}, func(n int) []string {
		if n == 1 {
			return failureSummary
		}
		return append([]string{"attempt 1"}, successSummary...)
	})

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 2, cs.logRequests)
}

func TestStreamBricklensCompletionRequiresOptIn(t *testing.T) {
	cs, srv := newCompletionServer(t, []string{runningRunBody, succeededRunBody}, func(int) []string {
		return successSummary
	})

	ok, _, err := streamForCompletion(t, srv.URL, false)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, cs.runGets, 2)
}
