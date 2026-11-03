package pipelines

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

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
		{"/Workspace/Users/someone@example.com/tests/test_x.py::test_x", "Workspace.Users.someone@example.com.tests.test_x", "test_x"},
	}
	for _, tc := range tests {
		className, name := junitCaseName(tc.testID)
		assert.Equal(t, tc.className, className, tc.testID)
		assert.Equal(t, tc.name, name, tc.testID)
	}
}

func TestWriteJUnitReport(t *testing.T) {
	// Acceptance tests cover each outcome. This covers what the test server does not send:
	// multi-line and truncated details, and text that XML must escape.
	const project = "/Workspace/Users/someone@example.com/.bundle/orders/dev/files"
	run := newTestRun()
	for _, e := range []testPipelineEvent{
		caseEvent(t, project+"/tests/test_a.py::test_bad", "FAILED", "",
			`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2\nwhere 1 = f()"}]}`,
			`"truncation": {"truncated_fields": [{"field_name": "error"}]}`),
		caseEvent(t, project+"/tests/test_a.py::test_setup", "ERROR", "",
			`"error": {"exceptions": [{"class_name": "RuntimeError", "message": "fixture <db> failed"}]}`),
	} {
		run.add(e)
	}

	path := filepath.Join(t.TempDir(), "report.xml")
	require.NoError(t, writeJUnitReport(path, newJUnitReport("orders", "u1", project, run, "")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="orders" tests="2" failures="1" errors="1" skipped="0">
    <testcase classname="tests.test_a" name="test_bad">
      <failure message="AssertionError: assert 1 == 2">AssertionError: assert 1 == 2&#xA;where 1 = f()&#xA;(details truncated)</failure>
    </testcase>
    <testcase classname="tests.test_a" name="test_setup">
      <error message="RuntimeError: fixture &lt;db&gt; failed">RuntimeError: fixture &lt;db&gt; failed</error>
    </testcase>
  </testsuite>
</testsuites>
`, string(data))
}

func TestTestReportJoinsWriteErrors(t *testing.T) {
	report := &testReport{path: filepath.Join(t.TempDir(), "missing", "report.xml"), out: io.Discard, suiteName: "orders"}
	runErr := errors.New("update u1 is FAILED")
	err := report.finish("u1", newTestRun(), runErr.Error(), runErr)
	require.ErrorIs(t, err, runErr)
	assert.ErrorContains(t, err, "failed to write JUnit report")
}
