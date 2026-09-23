package pipelines

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixtureResult reads the captured /events fixture, filters to the test
// event types (as the command does), and folds it into a testRunResult.
func loadFixtureResult(t *testing.T) *testRunResult {
	t.Helper()
	raw, err := os.ReadFile("testdata/pipeline_test_events.json")
	require.NoError(t, err)
	var resp struct {
		Events []testPipelineEvent `json:"events"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))
	return reduceTestEvents(filterTestEvents(resp.Events))
}

// assertGolden compares got against testdata/<name>, rewriting the golden file
// when UPDATE_GOLDEN=1 so the fixtures can be regenerated intentionally.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden %s; run with UPDATE_GOLDEN=1 to create it", name)
	assert.Equal(t, string(want), got)
}

func TestReduceTestEventsFolding(t *testing.T) {
	// Fixture is the DP emission /events payload in the flat-status proto shape:
	// PENDING -> RUNNING -> terminal per case, where the terminal status is
	// itself the verdict (PASSED/FAILED/SKIPPED/ERROR, enum-prefixed on the
	// wire) with a sibling result object carrying only terminal detail.
	r := loadFixtureResult(t)

	// All 4 discovered cases are present in first-seen (collection) order.
	require.Len(t, r.Cases, 4)
	assert.Equal(t, "test_demo.py::test_pass", r.Cases[0].NodeID)
	assert.Equal(t, "test_demo.py::test_fail", r.Cases[1].NodeID)
	assert.Equal(t, "test_demo.py::test_skip", r.Cases[2].NodeID)
	assert.Equal(t, "test_demo.py::test_error", r.Cases[3].NodeID)

	// Every case folded to a terminal state (PENDING/RUNNING collapsed).
	completed := r.completed()
	require.Len(t, completed, 4)

	// Terminal verdicts fold correctly onto status (enum prefix stripped).
	assert.Equal(t, testStatusPassed, r.Cases[0].Status)
	assert.Equal(t, testStatusFailed, r.Cases[1].Status)
	assert.Equal(t, testStatusSkipped, r.Cases[2].Status)
	assert.Equal(t, testStatusError, r.Cases[3].Status)

	// Identity (path/line) set at PENDING is carried through the fold.
	require.NotNil(t, r.Cases[1].Line)
	assert.Equal(t, "test_demo.py", r.Cases[1].Path)
	assert.Equal(t, 8, *r.Cases[1].Line)

	// The failing case surfaces a concise error line plus the full traceback.
	assert.Equal(t, "AssertionError: intentional demo failure", r.Cases[1].Message)
	assert.Contains(t, r.Cases[1].Traceback, "AssertionError: intentional demo failure")

	// The summary is the last test_summary event.
	require.NotNil(t, r.Summary)
	assert.Equal(t, testSummary{Total: 4, Passed: 1, Failed: 1, Skipped: 1, Errors: 1}, *r.Summary)
}

func TestRenderTestResultsTextGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	require.NoError(t, renderTestResultsText(&buf, "2d0401ce-8229-4795-926f-c3991369655e", "3627a3a1-2374-4bc2-981f-49bb81ef662c", r))
	assertGolden(t, "test_results.txt", buf.String())
}

func TestRenderJUnitXMLGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	require.NoError(t, renderJUnitXML(&buf, "2d0401ce-8229-4795-926f-c3991369655e", "3627a3a1-2374-4bc2-981f-49bb81ef662c", r))
	assertGolden(t, "test_results.xml", buf.String())
}

func TestRenderTestResultsJSONGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(r.toJSONOutput("2d0401ce-8229-4795-926f-c3991369655e", "3627a3a1-2374-4bc2-981f-49bb81ef662c")))
	assertGolden(t, "test_results.json", buf.String())
}

func TestRenderTextAllPassing(t *testing.T) {
	// A run with no failures omits the Failures section entirely.
	dur := int64(120)
	line := 3
	r := &testRunResult{
		Cases: []testCaseProgress{{
			NodeID: "tests/test_ok.py::test_a", Path: "tests/test_ok.py", Line: &line,
			Status: testStatusPassed, DurationMs: &dur,
			Message: "", Traceback: "", Truncated: false,
		}},
		Summary: &testSummary{Total: 1, Passed: 1, Failed: 0, Skipped: 0, Errors: 0},
	}
	var buf bytes.Buffer
	require.NoError(t, renderTestResultsText(&buf, "p", "u", r))
	out := buf.String()
	assert.Contains(t, out, "PASS   tests/test_ok.py::test_a (0.12s)")
	assert.NotContains(t, out, "Failures:")
	assert.Contains(t, out, "1 tests: 1 passed, 0 failed, 0 skipped, 0 errors")
}

func TestFormatTextDuration(t *testing.T) {
	zero := int64(0)
	one := int64(1)
	hundredTwenty := int64(120)

	assert.Equal(t, "<0.01s", formatTextDuration(&zero))
	assert.Equal(t, "<0.01s", formatTextDuration(&one))
	assert.Equal(t, "0.12s", formatTextDuration(&hundredTwenty))
}

func TestProgressPrinterStreamsTransitionsOnce(t *testing.T) {
	raw, err := os.ReadFile("testdata/pipeline_test_events.json")
	require.NoError(t, err)
	var resp struct {
		Events []testPipelineEvent `json:"events"`
	}
	require.NoError(t, json.Unmarshal(raw, &resp))

	var buf bytes.Buffer
	printer := newTestProgressPrinter(&buf, "pipeline-id", "update-id")
	require.NoError(t, printer.consume(resp.Events[:7]))
	firstSnapshot := buf.String()
	assert.Contains(t, firstSnapshot, "RUN    test_demo.py::test_pass")
	assert.Contains(t, firstSnapshot, "PASS   test_demo.py::test_pass")
	assert.Contains(t, firstSnapshot, "RUN    test_demo.py::test_fail")
	assert.NotContains(t, firstSnapshot, "FAIL   test_demo.py::test_fail")

	require.NoError(t, printer.consume(resp.Events))
	require.NoError(t, printer.consume(resp.Events))

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, "Pipeline test progress"))
	assert.Equal(t, 4, strings.Count(out, "RUN    "))
	assert.Equal(t, 1, strings.Count(out, "PASS   test_demo.py::test_pass"))
	assert.Equal(t, 1, strings.Count(out, "FAIL   test_demo.py::test_fail"))
	assert.Equal(t, 1, strings.Count(out, "SKIP   test_demo.py::test_skip"))
	assert.Equal(t, 1, strings.Count(out, "ERROR  test_demo.py::test_error"))
	assert.NotContains(t, out, "PENDING")
}
