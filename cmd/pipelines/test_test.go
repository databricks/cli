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
	req := buildStartTestUpdateRequest(testUpdateOptions{})
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test_only":true}`, string(b))
}

func TestBuildStartTestUpdateRequestWithSelectionAndFullRefresh(t *testing.T) {
	// Selectors may be paths or node ids; both fold into the single test_selection
	// collection (there is no separate test_files field anymore).
	req := buildStartTestUpdateRequest(testUpdateOptions{
		testSelection: []string{"tests/a.py", "tests/b/"},
		fullRefresh:   true,
	})
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test_only":true,"full_refresh":true,"test_selection":["tests/a.py","tests/b/"]}`, string(b))
}

func TestBuildStartTestUpdateRequestWithSelectionAndPytestArgs(t *testing.T) {
	// --select node ids populate test_selection; post-`--` args populate pytest_args.
	req := buildStartTestUpdateRequest(testUpdateOptions{
		testSelection: []string{"tests/test_orders.py::test_valid", "tests/test_orders.py::test_backfill"},
		pytestArgs:    []string{"-k", "smoke"},
	})
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test_only":true,"test_selection":["tests/test_orders.py::test_valid","tests/test_orders.py::test_backfill"],"pytest_args":["-k","smoke"]}`, string(b))
}

func TestCombineSelectionFoldsTestsAlias(t *testing.T) {
	// --select and the deprecated --tests alias fold into one selection list;
	// parseTestArgs then de-duplicates it.
	combined := combineSelection([]string{"a::t1", "tests/b.py"}, []string{"tests/b.py", "c::t3"})
	assert.Equal(t, []string{"a::t1", "tests/b.py", "tests/b.py", "c::t3"}, combined)
	assert.Equal(t, []string{"a::t1", "tests/b.py", "c::t3"}, unionNodeIDs(combined))
}

func TestSelectFlagWiredWithShorthand(t *testing.T) {
	// -s is not a global shorthand, so --select carries it.
	cmd := testCommand()
	f := cmd.Flags().ShorthandLookup("s")
	require.NotNil(t, f, "expected -s shorthand to be registered")
	assert.Equal(t, "select", f.Name)
}

func TestTestsFlagIsHiddenDeprecatedAlias(t *testing.T) {
	cmd := testCommand()
	f := cmd.Flags().Lookup("tests")
	require.NotNil(t, f, "expected --tests alias to still be defined")
	assert.True(t, f.Hidden, "expected --tests to be hidden from help")
	assert.NotEmpty(t, f.Deprecated, "expected --tests to carry a deprecation message")
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
			NodeID: "n", Path: "p", Line: nil, Status: testStatusPassed,
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

func TestSplitPipelineTestArgs(t *testing.T) {
	// No `--`: everything is positional, pytest passthrough is empty.
	pos, pytest := splitPipelineTestArgs([]string{"mypipe"}, -1)
	assert.Equal(t, []string{"mypipe"}, pos)
	assert.Empty(t, pytest)

	// `--` present (cobra strips it): dashPos counts the args before it. Here
	// args = ["mypipe","-k","smoke"] models `test mypipe -- -k smoke`.
	pos, pytest = splitPipelineTestArgs([]string{"mypipe", "-k", "smoke"}, 1)
	assert.Equal(t, []string{"mypipe"}, pos)
	assert.Equal(t, []string{"-k", "smoke"}, pytest)

	// `--` with no positional before it: `test -- -m slow`.
	pos, pytest = splitPipelineTestArgs([]string{"-m", "slow"}, 0)
	assert.Empty(t, pos)
	assert.Equal(t, []string{"-m", "slow"}, pytest)
}

func TestValidatePytestPassthrough(t *testing.T) {
	for _, ok := range [][]string{
		nil,
		{"-k", "smoke"},
		{"-m", "slow"},
		{"-ksmoke"},
		{"-mslow"},
		{"-k", "smoke", "-m", "slow"},
	} {
		assert.NoErrorf(t, validatePytestPassthrough(ok), "expected %v to be accepted", ok)
	}

	for _, bad := range [][]string{
		{"-x"},
		{"--maxfail=1"},
		{"smoke"}, // bare value with no -k/-m
		{"-k", "smoke", "-v"},
	} {
		assert.Errorf(t, validatePytestPassthrough(bad), "expected %v to be rejected", bad)
	}
}

func TestUnionNodeIDs(t *testing.T) {
	// Union across repeats de-duplicates while preserving first-seen order.
	got := unionNodeIDs([]string{"a::t1", "b::t2", "a::t1", "c::t3"})
	assert.Equal(t, []string{"a::t1", "b::t2", "c::t3"}, got)
}

func TestParseTestArgsMatrix(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		dashPos        int
		testSelection  []string
		exclude        []string
		pipelineIDFlag string
		wantKey        string
		wantIDFlag     string
		wantSelection  []string
		wantPytest     []string
		wantErr        string
	}{
		{
			name:    "no args -> infer (empty key, no id)",
			args:    nil,
			dashPos: -1,
		},
		{
			name:    "pipeline key only",
			args:    []string{"orders_pipeline"},
			dashPos: -1,
			wantKey: "orders_pipeline",
		},
		{
			name:          "pipeline key + one --select",
			args:          []string{"orders_pipeline"},
			dashPos:       -1,
			testSelection: []string{"tests/test_orders.py::test_valid"},
			wantKey:       "orders_pipeline",
			wantSelection: []string{"tests/test_orders.py::test_valid"},
		},
		{
			name:          "pipeline key + multiple --select (unioned, deduped)",
			args:          []string{"orders_pipeline"},
			dashPos:       -1,
			testSelection: []string{"a::t1", "b::t2", "a::t1"},
			wantKey:       "orders_pipeline",
			wantSelection: []string{"a::t1", "b::t2"},
		},
		{
			name:           "--pipeline-id + --select (no positional)",
			args:           nil,
			dashPos:        -1,
			testSelection:  []string{"a::t1"},
			pipelineIDFlag: "abc123",
			wantIDFlag:     "abc123",
			wantSelection:  []string{"a::t1"},
		},
		{
			name:          "--select with -- -k passthrough",
			args:          []string{"orders_pipeline", "-k", "smoke"},
			dashPos:       1,
			testSelection: []string{"a::t1"},
			wantKey:       "orders_pipeline",
			wantSelection: []string{"a::t1"},
			wantPytest:    []string{"-k", "smoke"},
		},
		{
			name:    "reject: two positionals (former variadic TEST_SELECTION)",
			args:    []string{"orders_pipeline", "tests/test_orders.py::test_valid"},
			dashPos: -1,
			wantErr: "accepts at most one PIPELINE_KEY",
		},
		{
			name:           "reject: PIPELINE_KEY + --pipeline-id together",
			args:           []string{"orders_pipeline"},
			dashPos:        -1,
			pipelineIDFlag: "abc123",
			wantErr:        "mutually exclusive",
		},
		{
			name:    "reject: unsupported pytest arg after --",
			args:    []string{"orders_pipeline", "--maxfail=1"},
			dashPos: 1,
			wantErr: "unsupported pytest argument",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTestArgs(tc.args, tc.dashPos, tc.testSelection, tc.exclude, tc.pipelineIDFlag)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantKey, got.PipelineKey)
			assert.Equal(t, tc.wantIDFlag, got.PipelineIDFlag)
			assert.Equal(t, tc.wantSelection, got.TestSelection)
			assert.Equal(t, tc.wantPytest, got.PytestArgs)
		})
	}
}

func TestResolvePipelineModeTable(t *testing.T) {
	// In-bundle, nothing given -> infer the sole pipeline (works with --select too).
	mode, err := resolvePipelineMode(true, parsedTestArgs{TestSelection: []string{"a::t1"}})
	require.NoError(t, err)
	assert.Equal(t, resolveByInference, mode)

	// In-bundle, PIPELINE_KEY given -> resolve by key.
	mode, err = resolvePipelineMode(true, parsedTestArgs{PipelineKey: "orders_pipeline"})
	require.NoError(t, err)
	assert.Equal(t, resolveByKey, mode)

	// --pipeline-id given -> resolve by id, in or out of a bundle.
	for _, inBundle := range []bool{true, false} {
		mode, err = resolvePipelineMode(inBundle, parsedTestArgs{PipelineIDFlag: "abc123"})
		require.NoError(t, err)
		assert.Equal(t, resolveByID, mode)
	}

	// Out of bundle, no --pipeline-id -> error.
	_, err = resolvePipelineMode(false, parsedTestArgs{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--pipeline-id is required")
}
