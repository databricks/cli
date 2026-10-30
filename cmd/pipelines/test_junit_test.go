package pipelines

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitCaseName(t *testing.T) {
	tests := []struct {
		testID, className, name string
	}{
		{"tests/test_orders.py::test_backfill", "tests.test_orders", "test_backfill"},
		{"tests/test_orders.py::TestOrders::test_tax[a-b]", "tests.test_orders.TestOrders", "test_tax[a-b]"},
		// A parameter ID may contain "::".
		{"tests/test_a.py::test_cast[a::int]", "tests.test_a", "test_cast[a::int]"},
		{"tests/test_a.py::TestCast::test_cast[a::int]", "tests.test_a.TestCast", "test_cast[a::int]"},
		{"tests/test_orders.py", "", "tests.test_orders"},
		{"/Workspace/Shared/tests/test_x.py::test_x", "Workspace.Shared.tests.test_x", "test_x"},
	}
	for _, tc := range tests {
		className, name := junitCaseName(tc.testID)
		assert.Equal(t, tc.className, className, tc.testID)
		assert.Equal(t, tc.name, name, tc.testID)
	}
}

func TestWriteJUnitReport(t *testing.T) {
	run := newTestRun()
	for _, e := range []testPipelineEvent{
		caseEvent(t, "tests/test_a.py::test_ok", "RUNNING", "2026-01-01T00:00:00.000Z"),
		caseEvent(t, "tests/test_a.py::test_ok", "PASSED", "2026-01-01T00:00:01.840Z"),
		caseEvent(t, "tests/test_a.py::test_bad", "FAILED", "",
			`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2\nwhere 1 = f()"}]}`,
			`"truncation": {"truncated_fields": [{"field_name": "error"}]}`),
		caseEvent(t, "tests/test_a.py::test_setup", "ERROR", "",
			`"error": {"exceptions": [{"class_name": "RuntimeError", "message": "fixture <db> failed"}]}`),
		caseEvent(t, "tests/test_a.py::test_old", "SKIPPED", ""),
		caseEvent(t, "tests/test_a.py::test_unfinished", "RUNNING", ""),
	} {
		run.add(e)
	}

	path := filepath.Join(t.TempDir(), "report.xml")
	require.NoError(t, os.WriteFile(path, []byte("previous report"), 0o644))
	require.NoError(t, writeJUnitReport(path, newJUnitReport("orders", "u1", run, "RUN INCOMPLETE: received 4 of 5 test results.")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="orders" tests="6" failures="1" errors="3" skipped="1">
    <testcase classname="tests.test_a" name="test_ok" time="1.840"></testcase>
    <testcase classname="tests.test_a" name="test_bad">
      <failure message="AssertionError: assert 1 == 2">AssertionError: assert 1 == 2&#xA;where 1 = f()&#xA;(details truncated)</failure>
    </testcase>
    <testcase classname="tests.test_a" name="test_setup">
      <error message="RuntimeError: fixture &lt;db&gt; failed">RuntimeError: fixture &lt;db&gt; failed</error>
    </testcase>
    <testcase classname="tests.test_a" name="test_old">
      <skipped></skipped>
    </testcase>
    <testcase classname="tests.test_a" name="test_unfinished">
      <error message="no result received"></error>
    </testcase>
    <testcase name="update u1">
      <error message="RUN INCOMPLETE: received 4 of 5 test results.">RUN INCOMPLETE: received 4 of 5 test results.</error>
    </testcase>
  </testsuite>
</testsuites>
`, string(data))
}

func TestTestRunProblem(t *testing.T) {
	empty := newTestRun()
	empty.add(summaryEvent(t, 0, 0, 0, 0, 0))
	assert.Equal(t, "No tests were collected.", testRunProblem(empty, "", root.ErrAlreadyPrinted))

	failed := newTestRun()
	failed.add(caseEvent(t, "t::a", "FAILED", ""))
	failed.add(summaryEvent(t, 1, 0, 1, 0, 0))
	assert.Empty(t, testRunProblem(failed, "", root.ErrAlreadyPrinted))

	passed := newTestRun()
	passed.add(caseEvent(t, "t::a", "PASSED", ""))
	passed.add(summaryEvent(t, 1, 1, 0, 0, 0))
	assert.Equal(t, "update u1 is FAILED", testRunProblem(passed, "", errors.New("update u1 is FAILED")))

	assert.Equal(t, "RUN INCOMPLETE: the update was canceled before all test results were received.\nupdate u1 is CANCELED",
		testRunProblem(newTestRun(), "the update was canceled before all test results were received", errors.New("update u1 is CANCELED")))
}

func TestTestReportJoinsWriteErrors(t *testing.T) {
	report := &testReport{path: filepath.Join(t.TempDir(), "missing", "report.xml"), out: io.Discard, suiteName: "orders"}
	runErr := errors.New("update u1 is FAILED")
	err := report.finish("u1", newTestRun(), runErr.Error(), runErr)
	require.ErrorIs(t, err, runErr)
	assert.ErrorContains(t, err, "failed to write JUnit report")
}

func TestTestReportReplacesAStaleReportOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.xml")
	require.NoError(t, os.WriteFile(path, []byte("<testsuites></testsuites>"), 0o644))
	report := &testReport{path: path, out: io.Discard, suiteName: "orders"}
	startErr := errors.New("An active update already exists.")
	assert.Equal(t, startErr, report.failed(t.Context(), startErr))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `<testcase name="pipelines test">`)
	assert.Contains(t, string(data), `<error message="An active update already exists.">`)
}

func TestTestReportNamesAPrintedError(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	logdiag.LogDiag(ctx, diag.Diagnostic{Severity: diag.Error, Summary: "reference does not exist: ${var.missing}"})
	path := filepath.Join(t.TempDir(), "report.xml")
	report := &testReport{path: path, out: io.Discard, suiteName: "orders"}
	assert.Equal(t, root.ErrAlreadyPrinted, report.failed(ctx, root.ErrAlreadyPrinted))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `<error message="reference does not exist: ${var.missing}">`)

	// The sentinel would hide a joined write error, so the write error is returned alone.
	report.path = filepath.Join(t.TempDir(), "missing", "report.xml")
	err = report.failed(ctx, root.ErrAlreadyPrinted)
	assert.NotErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.ErrorContains(t, err, "failed to write JUnit report")
}
