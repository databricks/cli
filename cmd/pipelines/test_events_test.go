package pipelines

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvents builds the events of one test, each with its own event ID.
type testEvents struct {
	t *testing.T
	n int
}

func newTestEvents(t *testing.T) *testEvents {
	return &testEvents{t: t}
}

func (b *testEvents) testCase(id, status, timestamp string, extra ...string) testPipelineEvent {
	b.n++
	return decodeTestEvent(b.t, fmt.Sprintf(`{
		"id": "e%d", "event_type": "test_case_progress", "timestamp": %q,
		"origin": {"test_case_id": %q},
		"details": {"test_case_progress": {"status": %q}}%s
	}`, b.n, timestamp, id, status, strings.Join(append([]string{""}, extra...), ",")))
}

func (b *testEvents) summary(total, passed, failed, skipped, errored int) testPipelineEvent {
	b.n++
	return decodeTestEvent(b.t, fmt.Sprintf(`{
		"id": "e%d", "event_type": "test_run_summary",
		"details": {"test_run_summary": {"total": %d, "passed": %d, "failed": %d, "skipped": %d, "errored": %d}}
	}`, b.n, total, passed, failed, skipped, errored))
}

func decodeTestEvent(t *testing.T, data string) testPipelineEvent {
	var e testPipelineEvent
	require.NoError(t, json.Unmarshal([]byte(data), &e))
	return e
}

func TestTestRunAdd(t *testing.T) {
	ev := newTestEvents(t)
	run := newTestRun()
	events := []testPipelineEvent{
		ev.testCase("tests/test_a.py::test_ok", "PENDING", "2026-01-01T00:00:00.000Z"),
		ev.testCase("tests/test_a.py::test_ok", "RUNNING", "2026-01-01T00:00:01.000Z"),
		ev.testCase("tests/test_a.py::test_ok", "PASSED", "2026-01-01T00:00:02.840Z"),
		ev.testCase("tests/test_a.py::test_bad", "PIPELINE_TEST_CASE_STATUS_FAILED", "2026-01-01T00:00:03.000Z",
			`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2"}]}`,
			`"truncation": {"truncated_fields": [{"field_name": "error"}]}`),
		// A late RUNNING event does not undo the final outcome.
		ev.testCase("tests/test_a.py::test_bad", "RUNNING", "2026-01-01T00:00:02.000Z"),
		ev.summary(2, 1, 1, 0, 0),
	}

	var finished []string
	for _, e := range events {
		if c := run.add(e); c != nil {
			finished = append(finished, c.id)
		}
	}
	// Reading every page again adds nothing.
	for _, e := range events {
		assert.Nil(t, run.add(e))
	}

	assert.Equal(t, []string{"tests/test_a.py::test_ok", "tests/test_a.py::test_bad"}, finished)
	require.Len(t, run.cases, 2)

	ok := run.cases[0]
	assert.Equal(t, testCasePassed, ok.status)
	d, known := ok.duration()
	assert.True(t, known)
	assert.Equal(t, 1840*time.Millisecond, d)

	bad := run.cases[1]
	assert.Equal(t, testCaseFailed, bad.status)
	assert.True(t, bad.detailsTruncated)
	assert.Equal(t, "AssertionError: assert 1 == 2", errorDetailText(bad.errorDetail))
	d, known = bad.duration()
	assert.True(t, known)
	assert.Equal(t, time.Second, d)

	assert.Empty(t, run.incompleteReason(pipelines.UpdateInfoStateFailed))
}

func TestTestRunIncompleteReason(t *testing.T) {
	ev := newTestEvents(t)
	passed := func() testPipelineEvent {
		return ev.testCase("t::a", "PASSED", "")
	}
	tests := []struct {
		name   string
		events []testPipelineEvent
		state  pipelines.UpdateInfoState
		want   string
	}{
		{
			name:   "no summary after completion",
			events: []testPipelineEvent{passed()},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "the update ended without a test summary",
		},
		{
			name:   "two summaries",
			events: []testPipelineEvent{passed(), ev.summary(1, 1, 0, 0, 0), ev.summary(1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "received 2 test summaries instead of one",
		},
		{
			name:   "missing outcome",
			events: []testPipelineEvent{passed(), ev.testCase("t::b", "RUNNING", ""), ev.summary(2, 2, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "received 1 of 2 test results",
		},
		{
			name:   "case without outcome that the summary leaves out",
			events: []testPipelineEvent{passed(), ev.testCase("t::b", "RUNNING", ""), ev.summary(1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "test t::b has no result",
		},
		{
			name:   "two outcomes for one case",
			events: []testPipelineEvent{passed(), ev.testCase("t::a", "FAILED", ""), ev.summary(1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "test t::a has more than one result",
		},
		{
			name:   "outcomes differ from summary",
			events: []testPipelineEvent{ev.testCase("t::a", "FAILED", ""), ev.summary(1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "the test results do not match the test summary",
		},
		{
			name: "outcome without details",
			events: []testPipelineEvent{
				passed(),
				decodeTestEvent(t, `{"id": "truncated", "event_type": "test_case_progress", "origin": {"test_case_id": "t::b"}}`),
				ev.summary(1, 1, 0, 0, 0),
			},
			state: pipelines.UpdateInfoStateCompleted,
			want:  "some test results could not be read",
		},
		{
			name:   "unknown status",
			events: []testPipelineEvent{passed(), ev.testCase("t::b", "PIPELINE_TEST_CASE_STATUS_UNSPECIFIED", ""), ev.summary(1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "some test results could not be read",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := newTestRun()
			for _, e := range tc.events {
				run.add(e)
			}
			assert.Equal(t, tc.want, run.incompleteReason(tc.state))
		})
	}
}

func TestReadTestEventsFollowsPages(t *testing.T) {
	server := testserver.New(t)
	var queries []url.Values
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/events", func(req testserver.Request) any {
		queries = append(queries, req.URL.Query())
		if req.URL.Query().Get("page_token") == "" {
			return map[string]any{"events": []map[string]string{{"id": "e1"}}, "next_page_token": "page2"}
		}
		return map[string]any{"events": []map[string]string{{"id": "e2"}}}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	events, err := readTestEvents(t.Context(), w, "p1", "u1", "2026-01-01T00:00:01.000Z")
	require.NoError(t, err)

	var ids []string
	for _, e := range events {
		ids = append(ids, e.Id)
	}
	assert.Equal(t, []string{"e1", "e2"}, ids)
	assert.Equal(t, []url.Values{
		{
			"filter":      {"update_id = 'u1' AND timestamp >= '2026-01-01T00:00:01.000Z'"},
			"order_by":    {"timestamp asc"},
			"max_results": {"250"},
		},
		{
			"page_token":  {"page2"},
			"max_results": {"250"},
		},
	}, queries)
}

// NFR3: a run of 200 cases spans several pages, and each case is counted once.
func TestReadTestEvents200Cases(t *testing.T) {
	const cases = 200
	var all []map[string]any
	for i := range cases {
		id := fmt.Sprintf("/Workspace/p/tests/test_many.py::test_case_%03d", i)
		for j, status := range []string{"PENDING", "RUNNING", "PASSED"} {
			all = append(all, map[string]any{
				"id": fmt.Sprintf("c%d-%d", i, j), "event_type": "test_case_progress", "timestamp": "2026-01-01T00:00:00.000Z",
				"origin": map[string]any{"test_case_id": id}, "details": map[string]any{"test_case_progress": map[string]any{"status": status}},
			})
		}
	}
	all = append(all, map[string]any{
		"id": "summary", "event_type": "test_run_summary",
		"details": map[string]any{"test_run_summary": map[string]any{"total": cases, "passed": cases}},
	})

	server := testserver.New(t)
	pages := 0
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/events", func(req testserver.Request) any {
		pages++
		offset := 0
		if token := req.URL.Query().Get("page_token"); token != "" {
			_, err := fmt.Sscan(token, &offset)
			require.NoError(t, err)
		}
		end := min(offset+testEventsPageSize, len(all))
		page := map[string]any{"events": all[offset:end]}
		if end < len(all) {
			page["next_page_token"] = strconv.Itoa(end)
		}
		return page
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	events, err := readTestEvents(t.Context(), w, "p1", "u1", "")
	require.NoError(t, err)
	assert.Equal(t, 3, pages)

	run := newTestRun()
	for _, e := range append(events, events...) {
		run.add(e)
	}
	assert.Empty(t, run.incompleteReason(pipelines.UpdateInfoStateCompleted))
	assert.Equal(t, testRunSummary{Total: cases, Passed: cases}, run.counts())
}
