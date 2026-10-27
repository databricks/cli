package pipelines

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEventSeq int

func caseEvent(t *testing.T, id, status, timestamp string, extra ...string) testPipelineEvent {
	testEventSeq++
	return decodeTestEvent(t, fmt.Sprintf(`{
		"id": "e%d", "event_type": "test_case_progress", "timestamp": %q,
		"origin": {"test_case_id": %q},
		"details": {"test_case_progress": {"status": %q}}%s
	}`, testEventSeq, timestamp, id, status, strings.Join(append([]string{""}, extra...), ",")))
}

func summaryEvent(t *testing.T, total, passed, failed, skipped, errors int) testPipelineEvent {
	testEventSeq++
	return decodeTestEvent(t, fmt.Sprintf(`{
		"id": "e%d", "event_type": "test_run_summary",
		"details": {"test_run_summary": {"total": %d, "passed": %d, "failed": %d, "skipped": %d, "errors": %d}}
	}`, testEventSeq, total, passed, failed, skipped, errors))
}

func decodeTestEvent(t *testing.T, data string) testPipelineEvent {
	var e testPipelineEvent
	require.NoError(t, json.Unmarshal([]byte(data), &e))
	return e
}

func TestTestRunAdd(t *testing.T) {
	run := newTestRun("")
	events := []testPipelineEvent{
		caseEvent(t, "tests/test_a.py::test_ok", "PENDING", "2026-01-01T00:00:00.000Z"),
		caseEvent(t, "tests/test_a.py::test_ok", "RUNNING", "2026-01-01T00:00:01.000Z"),
		caseEvent(t, "tests/test_a.py::test_ok", "PASSED", "2026-01-01T00:00:02.840Z"),
		caseEvent(t, "tests/test_a.py::test_bad", "PIPELINE_TEST_CASE_STATUS_FAILED", "2026-01-01T00:00:03.000Z",
			`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2"}]}`,
			`"truncation": {"truncated_fields": [{"field_name": "error"}]}`),
		// A late RUNNING event does not undo the final outcome.
		caseEvent(t, "tests/test_a.py::test_bad", "RUNNING", "2026-01-01T00:00:02.000Z"),
		summaryEvent(t, 2, 1, 1, 0, 0),
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
	passed := func() testPipelineEvent {
		return caseEvent(t, "t::a", "PASSED", "")
	}
	tests := []struct {
		name   string
		events []testPipelineEvent
		state  pipelines.UpdateInfoState
		want   string
	}{
		{
			name:   "no summary after failure",
			events: []testPipelineEvent{passed()},
			state:  pipelines.UpdateInfoStateFailed,
			want:   "the update failed before all test results were received",
		},
		{
			name:   "no summary after cancel",
			events: nil,
			state:  pipelines.UpdateInfoStateCanceled,
			want:   "the update was canceled before all test results were received",
		},
		{
			name:   "no summary after completion",
			events: []testPipelineEvent{passed()},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "the update ended without a test summary",
		},
		{
			name:   "two summaries",
			events: []testPipelineEvent{passed(), summaryEvent(t, 1, 1, 0, 0, 0), summaryEvent(t, 1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "received 2 test summaries instead of one",
		},
		{
			name:   "missing outcome",
			events: []testPipelineEvent{passed(), caseEvent(t, "t::b", "RUNNING", ""), summaryEvent(t, 2, 2, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "received 1 of 2 test results",
		},
		{
			name:   "two outcomes for one case",
			events: []testPipelineEvent{passed(), caseEvent(t, "t::a", "FAILED", ""), summaryEvent(t, 1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "test t::a has more than one result",
		},
		{
			name:   "outcomes differ from summary",
			events: []testPipelineEvent{caseEvent(t, "t::a", "FAILED", ""), summaryEvent(t, 1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "the test results do not match the test summary",
		},
		{
			name: "outcome without details",
			events: []testPipelineEvent{
				passed(),
				decodeTestEvent(t, `{"id": "truncated", "event_type": "test_case_progress", "origin": {"test_case_id": "t::b"}}`),
				summaryEvent(t, 1, 1, 0, 0, 0),
			},
			state: pipelines.UpdateInfoStateCompleted,
			want:  "some test results could not be read",
		},
		{
			name:   "unknown status",
			events: []testPipelineEvent{passed(), caseEvent(t, "t::b", "PIPELINE_TEST_CASE_STATUS_UNSPECIFIED", ""), summaryEvent(t, 1, 1, 0, 0, 0)},
			state:  pipelines.UpdateInfoStateCompleted,
			want:   "some test results could not be read",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := newTestRun("")
			for _, e := range tc.events {
				run.add(e)
			}
			assert.Equal(t, tc.want, run.incompleteReason(tc.state))
		})
	}
}

func TestTestRunFatalError(t *testing.T) {
	run := newTestRun("")
	run.add(decodeTestEvent(t, `{
		"id": "fatal", "event_type": "update_progress", "level": "ERROR",
		"error": {"fatal": true, "exceptions": [{"class_name": "ComputeTerminated", "message": "Compute stopped."}]}
	}`))
	assert.Equal(t, "ComputeTerminated: Compute stopped.", run.fatalError)
}

func TestTestEventReaderFollowsPages(t *testing.T) {
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
	reader, err := newTestEventReader(w, "p1", "u1")
	require.NoError(t, err)

	events, err := reader.read(t.Context(), "2026-01-01T00:00:01.000Z")
	require.NoError(t, err)

	var ids []string
	for _, e := range events {
		ids = append(ids, e.Id)
	}
	assert.Equal(t, []string{"e1", "e2"}, ids)
	assert.Equal(t, []url.Values{
		{
			"filter":      {"update_id='u1' AND timestamp >= '2026-01-01T00:00:01.000Z'"},
			"order_by":    {"timestamp asc"},
			"max_results": {"250"},
		},
		{
			"page_token":  {"page2"},
			"max_results": {"250"},
		},
	}, queries)
}

func TestTestRunCaseName(t *testing.T) {
	run := newTestRun("/Workspace/Users/someone/orders")
	assert.Equal(t, "tests/test_orders.py::test_a", run.caseName("/Workspace/Users/someone/orders/tests/test_orders.py::test_a"))
	assert.Equal(t, "/Workspace/Users/someone/orders2/tests/test_x.py::test_x", run.caseName("/Workspace/Users/someone/orders2/tests/test_x.py::test_x"))
	assert.Equal(t, "/Workspace/Shared/test_x.py::test_x", run.caseName("/Workspace/Shared/test_x.py::test_x"))
	assert.Equal(t, "/Workspace/Shared/test_x.py::test_x", newTestRun("").caseName("/Workspace/Shared/test_x.py::test_x"))
}
