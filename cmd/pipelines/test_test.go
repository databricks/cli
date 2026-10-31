package pipelines

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/root"
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
		"/Workspace/Users/someone/tests/test_x.py::test_x",
	}, resolved)

	_, err = resolveTestSelections([]string{"::test_x"}, toWorkspacePath)
	assert.EqualError(t, err, `test selection "::test_x" has no path`)

	_, err = resolveTestSelections([]string{"missing::test_x"}, toWorkspacePath)
	assert.EqualError(t, err, `cannot resolve test selection "missing::test_x": not found`)
}

func TestBundleShortTestID(t *testing.T) {
	syncRoot := t.TempDir()
	filesRoot := "/Workspace/Users/someone/.bundle/orders/dev/files"
	id := filesRoot + "/tests/test_a.py::TestA::test_x[a::b]"

	// The local path, relative to where the user stands, selects the test again.
	assert.Equal(t, "tests/test_a.py::TestA::test_x[a::b]", bundleShortTestID(syncRoot, syncRoot, filesRoot, id))
	assert.Equal(t, "test_a.py::TestA::test_x[a::b]", bundleShortTestID(filepath.Join(syncRoot, "tests"), syncRoot, filesRoot, id))
	assert.Equal(t, "../tests/test_a.py::TestA::test_x[a::b]", bundleShortTestID(filepath.Join(syncRoot, "src"), syncRoot, filesRoot, id))
	assert.Equal(t, "tests", bundleShortTestID(syncRoot, syncRoot, filesRoot, filesRoot+"/tests"))

	// Other IDs are shown as the service sends them.
	for _, other := range []string{
		"/Workspace/Users/someone/tests/test_s.py::test_s",
		filesRoot + "2/tests/test_a.py::test_x",
		"tests/test_a.py::test_x",
	} {
		assert.Equal(t, other, bundleShortTestID(syncRoot, syncRoot, filesRoot, other))
	}
	assert.Equal(t, id, bundleShortTestID(syncRoot, syncRoot, "", id))
}

func TestRootPathShortTestID(t *testing.T) {
	rootPath := "/Workspace/Users/someone/orders"
	assert.Equal(t, "tests/test_a.py::test_x", rootPathShortTestID(rootPath, rootPath+"/tests/test_a.py::test_x"))
	assert.Equal(t, "tests/test_a.py::test_x", rootPathShortTestID(rootPath+"/", rootPath+"/tests/test_a.py::test_x"))
	// The service accepts workspace paths with or without the /Workspace prefix.
	assert.Equal(t, "tests/test_a.py::test_x", rootPathShortTestID("/Users/someone/orders", rootPath+"/tests/test_a.py::test_x"))
	for _, other := range []string{
		"/Workspace/Users/someone/shared/test_s.py::test_s",
		rootPath + "2/test_a.py::test_x",
		"tests/test_a.py::test_x",
	} {
		assert.Equal(t, other, rootPathShortTestID(rootPath, other))
	}
	assert.Equal(t, rootPath+"/tests/test_a.py::test_x", rootPathShortTestID("", rootPath+"/tests/test_a.py::test_x"))
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

func TestTestTargetResumeCommand(t *testing.T) {
	tests := []struct {
		name   string
		target testTarget
		flags  map[string]string
		want   string
	}{
		{
			name:   "pipeline key",
			target: testTarget{key: "my_pipeline", pipelineID: "p1"},
			want:   "databricks pipelines test my_pipeline --resume u1",
		},
		{
			name:   "pipeline key with target and profile",
			target: testTarget{key: "my_pipeline", pipelineID: "p1"},
			flags:  map[string]string{"target": "dev", "profile": "staging"},
			want:   "databricks pipelines test my_pipeline --target dev --profile staging --resume u1",
		},
		{
			name:   "names that need quoting",
			target: testTarget{key: "my_pipeline", pipelineID: "p1"},
			flags:  map[string]string{"target": "dev west", "profile": "CI West"},
			want:   "databricks pipelines test my_pipeline --target 'dev west' --profile 'CI West' --resume u1",
		},
		{
			name:   "pipeline ID with profile",
			target: testTarget{pipelineID: "p1"},
			flags:  map[string]string{"profile": "staging"},
			want:   "databricks pipelines test --pipeline-id p1 --profile staging --resume u1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "databricks"}
			rootCmd.PersistentFlags().StringP("profile", "p", "", "")
			rootCmd.PersistentFlags().StringP("target", "t", "", "")
			pipelinesCmd := &cobra.Command{Use: "pipelines"}
			cmd := &cobra.Command{Use: "test"}
			rootCmd.AddCommand(pipelinesCmd)
			pipelinesCmd.AddCommand(cmd)
			for name, value := range tc.flags {
				require.NoError(t, cmd.Flag(name).Value.Set(value))
			}
			assert.Equal(t, tc.want, tc.target.resumeCommand(cmd, "u1"))
		})
	}
}

func TestTestRunError(t *testing.T) {
	tests := []struct {
		name    string
		events  func() []testPipelineEvent
		state   pipelines.UpdateInfoState
		wantErr string
	}{
		{
			name: "passed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{caseEvent(t, "t::a", "PASSED", ""), summaryEvent(t, 1, 1, 0, 0, 0)}
			},
			state: pipelines.UpdateInfoStateCompleted,
		},
		{
			name: "incomplete after completion",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{caseEvent(t, "t::a", "PASSED", "")}
			},
			state:   pipelines.UpdateInfoStateCompleted,
			wantErr: root.ErrAlreadyPrinted.Error(),
		},
		{
			name: "update failed with an error of its own and then the service's verdict",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					caseEvent(t, "t::a", "FAILED", ""),
					summaryEvent(t, 1, 0, 1, 0, 0),
					decodeTestEvent(t, `{"id": "own", "error": {"fatal": true, "exceptions": [{"message": "Commit failed."}]}}`),
					decodeTestEvent(t, `{"id": "verdict", "error": {"fatal": true, "exceptions": [{"class_name": "TestFailure", "message": "[PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed."}]}}`),
				}
			},
			state:   pipelines.UpdateInfoStateFailed,
			wantErr: "update u1 is FAILED: Commit failed.",
		},
		{
			name: "update failed with the service's verdict although no test failed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					caseEvent(t, "t::a", "PASSED", ""),
					summaryEvent(t, 1, 1, 0, 0, 0),
					decodeTestEvent(t, `{"id": "fatal", "error": {"fatal": true, "exceptions": [{"message": "[PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed."}]}}`),
				}
			},
			state:   pipelines.UpdateInfoStateFailed,
			wantErr: "update u1 is FAILED: [PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed.",
		},
		{
			name: "canceled after tests failed",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{caseEvent(t, "t::a", "FAILED", ""), summaryEvent(t, 1, 0, 1, 0, 0)}
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
