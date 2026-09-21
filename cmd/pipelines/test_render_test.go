package pipelines

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	r := loadFixtureResult(t)

	// All 7 discovered cases are present in first-seen (collection) order.
	require.Len(t, r.Cases, 7)
	assert.Equal(t, "tests/test_daily_revenue.py::test_sums_by_day", r.Cases[0].NodeID)
	assert.Equal(t, "tests/test_isolation.py::test_schema_isolation", r.Cases[3].NodeID)
	assert.Equal(t, "tests/test_daily_revenue.py::test_partition_present[wed]", r.Cases[6].NodeID)

	// Every case folded to its terminal COMPLETED state (PENDING/RUNNING collapsed).
	completed := r.completed()
	require.Len(t, completed, 7)

	// Terminal outcomes fold correctly.
	assert.Equal(t, testResultPassed, r.Cases[0].Result)
	assert.Equal(t, testResultFailed, r.Cases[1].Result)
	assert.Equal(t, testResultSkipped, r.Cases[2].Result)
	assert.Equal(t, testResultError, r.Cases[3].Result)

	// The failing case retains its message and traceback.
	assert.Equal(t, "revenue mismatch: expected 1230, got 1234", r.Cases[1].Message)
	assert.Contains(t, r.Cases[1].Traceback, "AssertionError")

	// The summary is the last test_summary event.
	require.NotNil(t, r.Summary)
	assert.Equal(t, testSummary{Total: 7, Passed: 4, Failed: 1, Skipped: 1, Errors: 1}, *r.Summary)
}

func TestRenderTestResultsTextGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	require.NoError(t, renderTestResultsText(&buf, "orders-dev", "01f0a3c2b19d", r))
	assertGolden(t, "test_results.txt", buf.String())
}

func TestRenderJUnitXMLGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	require.NoError(t, renderJUnitXML(&buf, "orders-dev", "01f0a3c2b19d", r))
	assertGolden(t, "test_results.xml", buf.String())
}

func TestRenderTestResultsJSONGolden(t *testing.T) {
	r := loadFixtureResult(t)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(r.toJSONOutput("orders-dev", "01f0a3c2b19d")))
	assertGolden(t, "test_results.json", buf.String())
}

func TestRenderTextAllPassing(t *testing.T) {
	// A run with no failures omits the Failures section entirely.
	dur := int64(120)
	line := 3
	r := &testRunResult{
		Cases: []testCaseProgress{{
			NodeID: "tests/test_ok.py::test_a", Path: "tests/test_ok.py", Line: &line,
			Status: testStatusCompleted, Result: testResultPassed, DurationMs: &dur,
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
