package pipelines

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/testserver"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePytestArgs(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr string
	}{
		{args: nil},
		{args: []string{""}, wantErr: "--pytest-arg requires a non-empty value"},
		{args: []string{"-k", "orders and not slow", "-m", "smoke,fast"}},
		{args: []string{"-korders"}, wantErr: `pytest argument "-korders" must be split: pass --pytest-arg=-k --pytest-arg=orders`},
		{args: []string{"-k=orders and not slow"}, wantErr: `pytest argument "-k=orders and not slow" must be split: pass --pytest-arg=-k --pytest-arg='orders and not slow'`},
		{args: []string{"-k", " "}, wantErr: "pytest argument -k requires a non-empty value"},
		{args: []string{"-k", "-m"}, wantErr: `pytest argument -k requires a value, not "-m": pass it in the next --pytest-arg`},
		{args: []string{"-k"}, wantErr: "pytest argument -k requires a value: pass it in the next --pytest-arg"},
		{args: []string{"-x"}, wantErr: `unsupported pytest argument "-x": only -k and -m are supported`},
		{args: []string{"--collect-only"}, wantErr: `unsupported pytest argument "--collect-only": only -k and -m are supported`},
		{args: []string{"tests/test_orders.py"}, wantErr: `unsupported pytest argument "tests/test_orders.py": pass test paths and test IDs as TEST_SELECTION arguments`},
	}
	for _, tc := range tests {
		err := validatePytestArgs(tc.args)
		if tc.wantErr == "" {
			assert.NoError(t, err, tc.args)
		} else {
			assert.EqualError(t, err, tc.wantErr, tc.args)
		}
	}
}

func TestResolveTestSelections(t *testing.T) {
	toWorkspacePath := func(p string) (string, error) {
		if p == "missing" {
			return "", errors.New("not found")
		}
		return "/Workspace/root/" + p, nil
	}

	resolved, err := resolveTestSelections([]string{
		"tests",
		"tests/test_orders.py::TestOrders::test_backfill[a::b]",
		"/Workspace/Users/someone/../someone/tests/test_x.py::test_x",
	}, toWorkspacePath)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"/Workspace/root/tests",
		"/Workspace/root/tests/test_orders.py::TestOrders::test_backfill[a::b]",
		"/Workspace/Users/someone/../someone/tests/test_x.py::test_x",
	}, resolved)

	_, err = resolveTestSelections([]string{"::test_x"}, toWorkspacePath)
	assert.EqualError(t, err, `test selection "::test_x" has no path`)

	_, err = resolveTestSelections([]string{"missing::test_x"}, toWorkspacePath)
	assert.EqualError(t, err, `cannot resolve test selection "missing::test_x": not found`)
}

func TestNewStartTestUpdateRequest(t *testing.T) {
	assert.Equal(t, startTestUpdateRequest{TestOnly: true}, newStartTestUpdateRequest(nil, nil))
	assert.Equal(t, startTestUpdateRequest{
		TestOnly:    true,
		TestOptions: &testOptions{TestSelection: []string{"/Workspace/tests"}, PytestArgs: []string{"-k", "orders"}},
	}, newStartTestUpdateRequest([]string{"/Workspace/tests"}, []string{"-k", "orders"}))
}

func TestRootPathSelection(t *testing.T) {
	got, err := rootPathSelection("/Workspace/Users/someone/orders/", "../other/tests")
	require.NoError(t, err)
	// The path is not cleaned, so the service rejects ".." with its own error.
	assert.Equal(t, "/Workspace/Users/someone/orders/../other/tests", got)

	_, err = rootPathSelection("", "tests")
	assert.ErrorContains(t, err, "the pipeline has no root_path")
}

// Source-linked deployments run the source files, as TranslatePaths maps them.
func TestBundleFilesRootSourceLinked(t *testing.T) {
	b := &bundle.Bundle{SyncRootPath: "/Workspace/Users/someone/orders"}
	b.Config.Workspace.FilePath = "/Workspace/Users/someone/.bundle/orders/dev/files"
	b.Config.Workspace.SyncMode = config.SyncModeInPlace
	assert.Equal(t, "/Workspace/Users/someone/orders", bundleFilesRoot(b))
}

func TestBundleTestSelectionPath(t *testing.T) {
	syncRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(syncRoot, "pipeline", "tests"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(syncRoot, "conftest.py"), nil, 0o644))
	filesRoot := "/Workspace/Users/someone/.bundle/orders/dev/files"

	// A path may leave the working directory as long as it stays in the synchronized files.
	got, err := bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline", "tests"), syncRoot, filesRoot, filepath.Join("..", "..", "conftest.py"))
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/conftest.py", got)

	// A path that does not exist locally is an error, so that a typo fails before compute starts.
	_, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, "tests/test_a.py")
	assert.EqualError(t, err, filepath.ToSlash(filepath.Join(syncRoot, "tests", "test_a.py"))+" does not exist; pass a local path or an absolute workspace path")

	// A path outside the synchronized files is an error. The error shows forward slashes.
	_, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, filepath.Join("..", "tests"))
	assert.ErrorContains(t, err, "is outside the synchronized bundle files in "+filepath.ToSlash(syncRoot))
}

func TestNotPipelineKeyError(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	b := &bundle.Bundle{}
	b.Config.Resources.Pipelines = map[string]*resources.Pipeline{"orders": {}, "customers": {}}
	err := notPipelineKeyError(cmd, b, "tests/test_orders.py", []string{"tests/test_orders.py", "tests/test_x.py::test_a b"})
	assert.EqualError(t, err, `"tests/test_orders.py" is not a pipeline in this bundle (pipelines: customers, orders). Usage: test customers tests/test_orders.py 'tests/test_x.py::test_a b'`)

	err = notPipelineKeyError(cmd, &bundle.Bundle{}, "tests", []string{"tests"})
	assert.EqualError(t, err, `"tests" is not a pipeline in this bundle, and the bundle has no pipelines`)
}

func TestTestRunError(t *testing.T) {
	ev := newTestEvents(t)
	tests := []struct {
		name    string
		events  func() []testPipelineEvent
		state   pipelines.UpdateInfoState
		wantErr string
	}{
		{
			name: "incomplete after completion",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{ev.testCase("t::a", "PASSED", "")}
			},
			state:   pipelines.UpdateInfoStateCompleted,
			wantErr: root.ErrAlreadyPrinted.Error(),
		},
		{
			name: "update failed with the service's verdict although no test failed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					ev.testCase("t::a", "PASSED", ""),
					ev.summary(1, 1, 0, 0, 0),
					decodeTestEvent(t, `{"id": "fatal", "error": {"fatal": true, "exceptions": [{"message": "[PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed."}]}}`),
				}
			},
			state:   pipelines.UpdateInfoStateFailed,
			wantErr: "update u1 is FAILED: [PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed.",
		},
		{
			name: "tests failed although the update completed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{ev.testCase("t::a", "FAILED", ""), ev.summary(1, 0, 1, 0, 0)}
			},
			state:   pipelines.UpdateInfoStateCompleted,
			wantErr: root.ErrAlreadyPrinted.Error(),
		},
		{
			name: "no tests collected although the update completed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{ev.summary(0, 0, 0, 0, 0)}
			},
			state:   pipelines.UpdateInfoStateCompleted,
			wantErr: root.ErrAlreadyPrinted.Error(),
		},
		{
			name: "all tests passed but the completed update reported a fatal error",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					ev.testCase("t::a", "PASSED", ""),
					ev.summary(1, 1, 0, 0, 0),
					decodeTestEvent(t, `{"id": "fatal", "error": {"fatal": true, "exceptions": [{"message": "result delivery failed"}]}}`),
				}
			},
			state:   pipelines.UpdateInfoStateCompleted,
			wantErr: "update u1 is COMPLETED: result delivery failed",
		},
		{
			name: "all tests were skipped",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{ev.testCase("t::a", "SKIPPED", ""), ev.summary(1, 0, 0, 1, 0)}
			},
			state: pipelines.UpdateInfoStateCompleted,
		},
		{
			name: "canceled after tests failed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{ev.testCase("t::a", "FAILED", ""), ev.summary(1, 0, 1, 0, 0)}
			},
			state:   pipelines.UpdateInfoStateCanceled,
			wantErr: "update u1 is CANCELED",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := newTestRun()
			for _, e := range tc.events() {
				run.add(e)
			}
			err := testRunError("u1", tc.state, run, run.incompleteReason(tc.state))
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantErr)
			}
		})
	}
}

func TestRelativeTestID(t *testing.T) {
	id := "/Workspace/Users/someone/orders/tests/test_a.py::test_x"
	assert.Equal(t, "tests/test_a.py::test_x", relativeTestID("/Workspace/Users/someone/orders", id))
	// root_path may lack the /Workspace prefix that the service adds to test IDs.
	assert.Equal(t, "tests/test_a.py::test_x", relativeTestID("/Users/someone/orders/", id))
	assert.Equal(t, id, relativeTestID("/Workspace/Users/someone/other", id))
	assert.Equal(t, id, relativeTestID("", id))
}

// The wait reads events from the last timestamp, where a later event can share that timestamp,
// and reads all events again after the update ends, where an event can appear late.
func TestWaitForTestUpdate(t *testing.T) {
	event := func(id, caseID, status, timestamp string) map[string]any {
		return map[string]any{
			"id": id, "event_type": "test_case_progress", "timestamp": timestamp,
			"origin": map[string]any{"test_case_id": caseID}, "details": map[string]any{"test_case_progress": map[string]any{"status": status}},
		}
	}
	a, b := "/Workspace/p/tests/test_a.py::test_a", "/Workspace/p/tests/test_a.py::test_b"
	first := []map[string]any{
		event("e1", a, "PENDING", "2026-01-01T00:00:01.000Z"),
		event("e2", b, "PENDING", "2026-01-01T00:00:01.000Z"),
		event("e3", a, "PASSED", "2026-01-01T00:00:02.000Z"),
	}
	// e4 has the timestamp of the cursor; e5 and the summary are readable only after the update ends.
	tie := append(first[2:], event("e4", b, "RUNNING", "2026-01-01T00:00:02.000Z"))
	late := []map[string]any{event("e5", b, "PASSED", "2026-01-01T00:00:03.000Z"), {
		"id": "e6", "event_type": "test_run_summary", "timestamp": "2026-01-01T00:00:04.000Z",
		"details": map[string]any{"test_run_summary": map[string]any{"total": 2, "passed": 2}},
	}}

	server := testserver.New(t)
	polls := 0
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/updates/{update_id}", func(req testserver.Request) any {
		polls++
		state := "RUNNING"
		if polls == 2 {
			state = "COMPLETED"
		}
		return map[string]any{"update": map[string]any{"update_id": "u1", "state": state}}
	})
	var filters []string
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/events", func(req testserver.Request) any {
		filter := req.URL.Query().Get("filter")
		filters = append(filters, filter)
		switch {
		case !strings.Contains(filter, "timestamp") && polls == 1:
			return map[string]any{"events": first}
		case strings.Contains(filter, "timestamp"):
			return map[string]any{"events": tie}
		default:
			return map[string]any{"events": append(append(first, tie...), late...)}
		}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	var out bytes.Buffer
	run := newTestRun()
	printer := &testProgressPrinter{out: &out, shortID: func(id string) string { return relativeTestID("/Workspace/p", id) }}
	state, err := waitForTestUpdate(t.Context(), w, "p1", "u1", run, printer)
	require.NoError(t, err)

	assert.Equal(t, pipelines.UpdateInfoStateCompleted, state)
	assert.Equal(t, []string{
		"update_id = 'u1'",
		"update_id = 'u1' AND timestamp >= '2026-01-01T00:00:02.000Z'",
		"update_id = 'u1'",
	}, filters)
	assert.Empty(t, run.incompleteReason(state))
	assert.Equal(t, testRunSummary{Total: 2, Passed: 2}, run.counts())
	assert.Contains(t, out.String(), "2 of 2 PASSED tests/test_a.py::test_b")
}
