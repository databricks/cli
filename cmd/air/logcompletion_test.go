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

// withNode prefixes a node's training output to a summary.
func withNode(node int, summary []string) []string {
	return append([]string{"training node " + strconv.Itoa(node)}, summary...)
}

// logsBody renders a log endpoint response for one node's attempt. Records are
// timestamped one second apart from 1700000000s and keyed by attempt, position
// and content, so a poll that re-serves earlier lines dedups them while new or
// repeated lines (the summary's rules) stay distinct; newestFirst matches an
// ascending=false query.
func logsBody(t *testing.T, nodeCount, node int, attempt string, lines []string, newestFirst bool) string {
	t.Helper()
	type record struct {
		RecordID     string `json:"record_id"`
		TimeUnixNano int64  `json:"time_unix_nano"`
		Body         string `json:"body"`
		NodeIndex    int    `json:"node_index"`
	}
	records := make([]record, len(lines))
	for i, line := range lines {
		records[i] = record{
			RecordID:     strconv.Itoa(node) + ":" + attempt + ":" + strconv.Itoa(i) + ":" + line,
			TimeUnixNano: (1_700_000_000 + int64(i)) * 1_000_000_000,
			Body:         line,
			NodeIndex:    node,
		}
	}
	if newestFirst {
		slices.Reverse(records)
	}
	b, err := json.Marshal(map[string]any{"log_records": records, "node_count": nodeCount})
	assert.NoError(t, err)
	return string(b)
}

const (
	runningRunBody    = `{"run_id":1,"start_time":1700000000000,"state":{"life_cycle_state":"RUNNING"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	succeededRunBody  = `{"run_id":1,"start_time":1700000000000,"end_time":1700000100000,"state":{"life_cycle_state":"TERMINATED","result_state":"SUCCESS"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	failedRunBody     = `{"run_id":1,"start_time":1700000000000,"end_time":1700000100000,"state":{"life_cycle_state":"TERMINATED","result_state":"FAILED"},"tasks":[{"run_id":2,"attempt_number":0}]}`
	retryingRunBody   = `{"run_id":1,"start_time":1700000000000,"state":{"life_cycle_state":"RUNNING"},"tasks":[{"run_id":2,"attempt_number":0},{"run_id":3,"attempt_number":1}]}`
	retrySucceededRun = `{"run_id":1,"start_time":1700000000000,"end_time":1700000100000,"state":{"life_cycle_state":"TERMINATED","result_state":"SUCCESS"},"tasks":[{"run_id":2,"attempt_number":0},{"run_id":3,"attempt_number":1}]}`
)

// logQuery is one log request the completion server received. attempt is the raw
// ref.attempt_number ("" for latest) and resolved the attempt it was served for;
// tail is true for a newest-first (completion check) query.
type logQuery struct {
	node     int
	attempt  string
	resolved string
	tail     bool
}

// completionServer serves an active run for the completion tests. runBodies are
// returned by successive runs/get calls (the last repeats). latestAttempt, when
// set, resolves a query without an attempt number, as the endpoint does;
// otherwise it resolves to attempt 0.
type completionServer struct {
	runGets       int
	queries       []logQuery
	latestAttempt func() string
}

// tailAttempts returns the raw attempt numbers of the completion checks for a node.
func (cs *completionServer) tailAttempts(node int) []string {
	var out []string
	for _, q := range cs.queries {
		if q.node == node && q.tail {
			out = append(out, q.attempt)
		}
	}
	return out
}

// newCompletionServer starts the server; nodeLines returns the lines for a log
// request, given the requests received so far including it.
func newCompletionServer(t *testing.T, nodeCount int, runBodies []string, nodeLines func(q logQuery, cs *completionServer) []string) (*completionServer, *httptest.Server) {
	t.Helper()
	original := bricklensPollInterval
	bricklensPollInterval = time.Millisecond
	t.Cleanup(func() { bricklensPollInterval = original })

	cs := &completionServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/logs"):
			node, err := strconv.Atoi(r.URL.Query().Get("filter.node_index"))
			if !assert.NoError(t, err) {
				return
			}
			q := logQuery{
				node:     node,
				attempt:  r.URL.Query().Get("ref.attempt_number"),
				resolved: r.URL.Query().Get("ref.attempt_number"),
				tail:     r.URL.Query().Get("ascending") == "false",
			}
			if q.resolved == "" {
				q.resolved = "0"
				if cs.latestAttempt != nil {
					q.resolved = cs.latestAttempt()
				}
			}
			cs.queries = append(cs.queries, q)
			_, _ = w.Write([]byte(logsBody(t, nodeCount, node, q.resolved, nodeLines(q, cs), q.tail)))
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

func TestStreamBricklensExitsWhenAllNodesComplete(t *testing.T) {
	cs, srv := newCompletionServer(t, 2, []string{runningRunBody}, func(q logQuery, _ *completionServer) []string {
		return withNode(q.node, successSummary)
	})

	ok, out, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	assert.True(t, ok)
	// Exited on the first poll, without Jobs ever reporting the run terminal.
	assert.Equal(t, 0, cs.runGets)
	// Both nodes, the streamed one included, were checked for attempt 0.
	assert.Equal(t, []string{"0"}, cs.tailAttempts(0))
	assert.Equal(t, []string{"0"}, cs.tailAttempts(1))
	assert.Contains(t, out, `"line":"training node 0"`)
}

func TestStreamBricklensWaitsForEveryNode(t *testing.T) {
	cs, srv := newCompletionServer(t, 2, []string{runningRunBody}, func(q logQuery, cs *completionServer) []string {
		if q.node == 1 && len(cs.tailAttempts(1)) < 3 {
			return []string{"still training"}
		}
		return successSummary
	})

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, cs.tailAttempts(1), 3)
}

func TestStreamBricklensIgnoresFailureSummary(t *testing.T) {
	cs, srv := newCompletionServer(t, 1, []string{runningRunBody, failedRunBody}, func(q logQuery, _ *completionServer) []string {
		return failureSummary
	})

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	// Waited for Jobs: a failed attempt may still be retried.
	assert.False(t, ok)
	assert.GreaterOrEqual(t, cs.runGets, 2)
	assert.Empty(t, cs.tailAttempts(0))
}

func TestStreamBricklensCompletionRequiresOptIn(t *testing.T) {
	cs, srv := newCompletionServer(t, 1, []string{runningRunBody, succeededRunBody}, func(q logQuery, _ *completionServer) []string {
		return successSummary
	})

	ok, _, err := streamForCompletion(t, srv.URL, false)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, cs.runGets, 2)
	assert.Empty(t, cs.tailAttempts(0))
}

func TestStreamBricklensBindsCompletionToLatestAttempt(t *testing.T) {
	// Attempt 0 succeeds on node 0 but fails on node 1, and is retried. On attempt
	// 1, node 0 succeeds while node 1 is still running when Jobs reports the run
	// terminal. Every check must target the attempt that triggered it, so the
	// stream ends on Jobs' terminal state.
	attemptOneStreamPolls := 0
	cs, srv := newCompletionServer(t, 2, []string{retryingRunBody, retryingRunBody, retrySucceededRun}, func(q logQuery, _ *completionServer) []string {
		switch {
		case q.resolved == "0" && q.node == 0:
			return successSummary
		case q.resolved == "0" && q.node == 1:
			return failureSummary
		case q.node == 1:
			return []string{"still training"}
		case q.tail:
			return successSummary
		default:
			// Node 0's stream on attempt 1: the summary arrives on the second poll.
			attemptOneStreamPolls++
			if attemptOneStreamPolls >= 2 {
				return append([]string{"attempt 1 node 0"}, successSummary...)
			}
			return []string{"attempt 1 node 0"}
		}
	})
	cs.latestAttempt = func() string {
		if cs.runGets == 0 {
			return "0"
		}
		return "1"
	}

	ok, _, err := streamForCompletion(t, srv.URL, true)
	require.NoError(t, err)
	// Ended on Jobs' terminal SUCCESS, not on summaries from different attempts.
	assert.True(t, ok)
	assert.GreaterOrEqual(t, cs.runGets, 3)
	assert.Equal(t, []string{"0", "1"}, cs.tailAttempts(1))
}
