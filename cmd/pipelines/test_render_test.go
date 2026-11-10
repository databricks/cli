package pipelines

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func shortTestID(id string) string {
	return strings.TrimPrefix(id, "/Workspace/p/")
}

func TestTestProgressPrinter(t *testing.T) {
	var out bytes.Buffer
	p := &testProgressPrinter{out: &out, shortID: shortTestID}
	p.updateState(pipelines.UpdateInfoStateWaitingForResources)
	p.updateState(pipelines.UpdateInfoStateInitializing)
	p.updateState(pipelines.UpdateInfoStateRunning)

	ev := newTestEvents(t)
	run := newTestRun()
	run.add(ev.testCase("/Workspace/p/tests/test_orders.py::test_backfill", "RUNNING", "2026-01-01T00:00:00.000Z"))
	p.caseFinished(run.add(ev.testCase("/Workspace/p/tests/test_orders.py::test_backfill", "FAILED", "2026-01-01T00:00:02.140Z")), 2)
	long := "/Workspace/p/tests/" + strings.Repeat("x", 70) + ".py::test_y"
	p.caseFinished(run.add(ev.testCase(long, "SKIPPED", "")), 2)

	assert.Equal(t, `Waiting for compute...
Running tests...
1 of 2 FAILED tests/test_orders.py::test_backfill .............................. [FAILED in 2.14s]
2 of 2 SKIPPED `+shortTestID(long)+`  [SKIPPED]
`, out.String())
}

func TestRenderTestResult(t *testing.T) {
	ev := newTestEvents(t)
	render := func(run *testRun, incompleteReason string) string {
		var out bytes.Buffer
		renderTestResult(&out, run, incompleteReason, shortTestID)
		renderTestSummaryLine(&out, run, incompleteReason)
		return out.String()
	}

	failed := newTestRun()
	failed.add(ev.testCase("/Workspace/p/tests/test_a.py::test_ok", "PASSED", ""))
	failed.add(ev.testCase("/Workspace/p/tests/test_a.py::test_bad", "FAILED", "",
		`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2"}]}`,
		`"truncation": {"truncated_fields": [{"field_name": "error"}]}`))
	failed.add(ev.testCase("/Workspace/p/tests/test_a.py::test_fixture", "ERROR", ""))
	failed.add(ev.summary(3, 1, 1, 0, 1))
	assert.Equal(t, `
Completed with 1 failed and 1 errored tests:

Failure in tests/test_a.py::test_bad
  AssertionError: assert 1 == 2
  (details truncated)

Error in tests/test_a.py::test_fixture
  No details were reported.

Done. PASSED=1 FAILED=1 SKIPPED=0 ERROR=1 TOTAL=3
`, render(failed, ""))

	incomplete := newTestRun()
	incomplete.add(ev.testCase("/Workspace/p/tests/test_b.py::test_one", "PASSED", ""))
	incomplete.add(ev.testCase("/Workspace/p/tests/test_b.py::test_two", "PENDING", ""))
	assert.Equal(t, `
RUN INCOMPLETE: the update failed before all test results were received.
Done. PASSED=1 FAILED=0 SKIPPED=0 ERROR=0 TOTAL=2
`, render(incomplete, "the update failed before all test results were received"))

	empty := newTestRun()
	empty.add(ev.summary(0, 0, 0, 0, 0))
	assert.Equal(t, `
No tests were collected.
Done. PASSED=0 FAILED=0 SKIPPED=0 ERROR=0 TOTAL=0
`, render(empty, ""))
}
