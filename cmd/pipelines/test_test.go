package pipelines

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildStartTestUpdateRequestMarshalsTestOnly(t *testing.T) {
	// The core interim-approach guarantee: the StartUpdate body always carries
	// test_only=true, which the generated SDK cannot express.
	req := buildStartTestUpdateRequest(nil, false)
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test_only":true}`, string(b))
}

func TestBuildStartTestUpdateRequestWithTestsAndFullRefresh(t *testing.T) {
	req := buildStartTestUpdateRequest([]string{"tests/a.py", "tests/b/"}, true)
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test_only":true,"full_refresh":true,"test_files":["tests/a.py","tests/b/"]}`, string(b))
}

func TestIsTerminalUpdateState(t *testing.T) {
	terminal := []pipelines.UpdateInfoState{
		pipelines.UpdateInfoStateCompleted,
		pipelines.UpdateInfoStateFailed,
		pipelines.UpdateInfoStateCanceled,
	}
	for _, s := range terminal {
		assert.Truef(t, isTerminalUpdateState(s), "expected %s terminal", s)
	}

	nonTerminal := []pipelines.UpdateInfoState{
		pipelines.UpdateInfoStateCreated,
		pipelines.UpdateInfoStateInitializing,
		pipelines.UpdateInfoStateRunning,
		pipelines.UpdateInfoStateQueued,
		pipelines.UpdateInfoStateWaitingForResources,
	}
	for _, s := range nonTerminal {
		assert.Falsef(t, isTerminalUpdateState(s), "expected %s non-terminal", s)
	}
}

func TestFilterTestEvents(t *testing.T) {
	events := []testPipelineEvent{
		{EventType: "update_progress", Timestamp: "", Details: testEventDetails{TestCaseProgress: nil, TestSummary: nil}},
		{EventType: testCaseProgressEventType, Timestamp: "", Details: testEventDetails{TestCaseProgress: &testCaseProgress{NodeID: "n"}, TestSummary: nil}},
		{EventType: "flow_progress", Timestamp: "", Details: testEventDetails{TestCaseProgress: nil, TestSummary: nil}},
		{EventType: testSummaryEventType, Timestamp: "", Details: testEventDetails{TestCaseProgress: nil, TestSummary: &testSummary{Total: 1}}},
	}
	filtered := filterTestEvents(events)
	require.Len(t, filtered, 2)
	assert.Equal(t, testCaseProgressEventType, filtered[0].EventType)
	assert.Equal(t, testSummaryEventType, filtered[1].EventType)
}

func TestCheckCompleteNoSummary(t *testing.T) {
	r := &testRunResult{Cases: nil, Summary: nil}
	err := r.checkComplete()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PIPELINE_TEST_NO_RESULT")
}

func TestCheckCompleteIncomplete(t *testing.T) {
	r := &testRunResult{
		Cases: []testCaseProgress{{
			NodeID: "n", Path: "p", Line: nil, Status: testStatusCompleted, Result: testResultPassed,
			DurationMs: nil, Message: "", Traceback: "", Truncated: false,
		}},
		Summary: &testSummary{Total: 3, Passed: 1, Failed: 0, Skipped: 0, Errors: 0},
	}
	err := r.checkComplete()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PIPELINE_TEST_EVENTS_INCOMPLETE")
}

func TestCheckCompleteAndFailed(t *testing.T) {
	r := loadFixtureResult(t)
	require.NoError(t, r.checkComplete())
	// The fixture run has a failure and an error, so it must exit non-zero.
	assert.True(t, r.failed())
}

func TestFailedFalseWhenAllPass(t *testing.T) {
	r := &testRunResult{Cases: nil, Summary: &testSummary{Total: 2, Passed: 2, Failed: 0, Skipped: 0, Errors: 0}}
	assert.False(t, r.failed())
}
