package pipelines

import (
	"encoding/json"
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
		{args: []string{"-korders", "-msmoke"}},
		{args: []string{"-k", "-m"}},
		{args: []string{"-k"}, wantErr: "pytest argument -k requires a value: pass it in the next --pytest-arg"},
		{args: []string{"-m", "smoke", "-m"}, wantErr: "pytest argument -m requires a value: pass it in the next --pytest-arg"},
		{args: []string{"-x"}, wantErr: `unsupported pytest argument "-x": only -k and -m are supported`},
		{args: []string{"--maxfail=1"}, wantErr: `unsupported pytest argument "--maxfail=1": only -k and -m are supported`},
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

// TestValidatePytestArgsRejectsUnsupportedOptions covers each form of option that the API rejects.
func TestValidatePytestArgsRejectsUnsupportedOptions(t *testing.T) {
	for _, arg := range []string{"--collect-only", "-n", "--rootdir=/", "--junit-xml=out.xml"} {
		err := validatePytestArgs([]string{arg})
		assert.EqualError(t, err, `unsupported pytest argument "`+arg+`": only -k and -m are supported`)
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

func TestBundleWorkspacePath(t *testing.T) {
	syncRoot := t.TempDir()
	bundleRoot := filepath.Join(syncRoot, "orders")
	filePath := "/Workspace/Users/someone/.bundle/orders/dev/files"

	got, err := bundleWorkspacePath(bundleRoot, syncRoot, filePath, filepath.Join("tests", "test_orders.py"))
	require.NoError(t, err)
	assert.Equal(t, filePath+"/orders/tests/test_orders.py", got)

	got, err = bundleWorkspacePath(filepath.Join(bundleRoot, "tests"), syncRoot, filePath, filepath.Join("..", "conftest.py"))
	require.NoError(t, err)
	assert.Equal(t, filePath+"/orders/conftest.py", got)

	got, err = bundleWorkspacePath(syncRoot, syncRoot, filePath, ".")
	require.NoError(t, err)
	assert.Equal(t, filePath, got)

	_, err = bundleWorkspacePath(syncRoot, syncRoot, filePath, filepath.Join("..", "tests"))
	assert.ErrorContains(t, err, "is outside the synchronized bundle files in "+syncRoot)
}

func TestTestIDBase(t *testing.T) {
	tests := []struct {
		name     string
		rootPath string
		sources  []string
		want     string
	}{
		{
			name:     "deepest folder holding the root path and every test source",
			rootPath: "/Workspace/Users/u/files/src/my_pipeline",
			sources:  []string{"/Workspace/Users/u/files/tests/**", "/Workspace/Users/u/files/checks/test_x.py"},
			want:     "/Workspace/Users/u/files",
		},
		{
			name:     "root path that holds the test sources",
			rootPath: "/Users/u/sl",
			sources:  []string{"/Users/u/sl/tests/**"},
			want:     "/Workspace/Users/u/sl",
		},
		{
			name:    "without a root path, the folder of a file test source",
			sources: []string{"/Workspace/Users/u/tests/test_x.py"},
			want:    "/Workspace/Users/u/tests",
		},
		{
			name:    "without a root path, the deepest folder holding the folders of every test source",
			sources: []string{"/Workspace/Users/u/a/tests/**", "/Workspace/Users/u/b/tests/**"},
			want:    "/Workspace/Users/u",
		},
		{
			name:    "without a root path, the parent of a folder test source",
			sources: []string{"/Workspace/Users/u/tests/**"},
			want:    "/Workspace/Users/u",
		},
		{
			name:     "root path without test sources",
			rootPath: "/Workspace/Users/u/orders",
			want:     "/Workspace/Users/u/orders",
		},
		{
			name: "neither",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, testIDBase(tc.rootPath, tc.sources))
		})
	}
}

func TestTestIDBasePath(t *testing.T) {
	got, err := testIDBasePath("/Workspace/Users/someone/orders", "tests/test_orders.py")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/someone/orders/tests/test_orders.py", got)

	_, err = testIDBasePath("", "tests")
	assert.EqualError(t, err, "the pipeline has no root_path or testing.sources; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
}

func TestBundleFilesRoot(t *testing.T) {
	filePath := "/Workspace/Users/someone/.bundle/orders/dev/files"
	syncRoot := "/Workspace/Users/someone/orders"
	for _, tc := range []struct {
		mode config.SyncMode
		want string
	}{
		{mode: config.SyncModeOverwrite, want: filePath},
		// Source-linked deployments run the source files, as TranslatePaths maps them.
		{mode: config.SyncModeInPlace, want: syncRoot},
		// Immutable deployments run a snapshot, whose path the command does not know.
		{mode: config.SyncModeImmutable, want: ""},
	} {
		b := &bundle.Bundle{SyncRootPath: syncRoot}
		b.Config.Workspace.FilePath = filePath
		b.Config.Workspace.SyncMode = tc.mode
		assert.Equal(t, tc.want, bundleFilesRoot(b), tc.mode)
	}
}

func TestBundleTestSelectionPath(t *testing.T) {
	syncRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(syncRoot, "pipeline", "tests"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(syncRoot, "pipeline", "tests", "test_a.py"), nil, 0o644))
	filesRoot := "/Workspace/Users/someone/.bundle/orders/dev/files"
	base := filesRoot + "/pipeline"

	// A path that exists where the user stands is a local path.
	got, err := bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline"), syncRoot, filesRoot, base, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/pipeline/tests/test_a.py", got)

	// Otherwise it is a test ID, relative to the folder that test IDs are relative to.
	got, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, base, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/pipeline/tests/test_a.py", got)

	// Without that folder, it is a local path.
	got, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, "", "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/tests/test_a.py", got)

	// A path that leaves that folder is a local path too. The error shows forward slashes.
	for _, testIDBase := range []string{base, ""} {
		_, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, testIDBase, filepath.Join("..", "tests"))
		assert.ErrorContains(t, err, "is outside the synchronized bundle files in "+filepath.ToSlash(syncRoot))
	}

	// When the deployed path of local files is unknown, a local path is an error, but a test ID still works.
	_, err = bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline"), syncRoot, "", base, "tests/test_a.py")
	assert.EqualError(t, err, "local test paths are not supported with workspace.sync_mode immutable; pass the test ID that the results show or an absolute workspace path")
	got, err = bundleTestSelectionPath(syncRoot, syncRoot, "", base, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/pipeline/tests/test_a.py", got)
}

func TestNeedsTestIDBase(t *testing.T) {
	assert.False(t, needsTestIDBase(nil))
	assert.False(t, needsTestIDBase([]string{"/Workspace/tests", "/Workspace/a.py::test_x"}))
	assert.True(t, needsTestIDBase([]string{"/Workspace/tests", "tests/test_a.py::test_x"}))
}

func TestNewStartTestUpdateRequest(t *testing.T) {
	tests := []struct {
		selection  []string
		pytestArgs []string
		want       string
	}{
		{want: `{"test_only":true}`},
		{
			selection: []string{"/Workspace/tests"},
			want:      `{"test_only":true,"test_options":{"test_selection":["/Workspace/tests"]}}`,
		},
		{
			pytestArgs: []string{"-m", "smoke,fast"},
			want:       `{"test_only":true,"test_options":{"pytest_args":["-m","smoke,fast"]}}`,
		},
	}
	for _, tc := range tests {
		body, err := json.Marshal(newStartTestUpdateRequest(tc.selection, tc.pytestArgs))
		require.NoError(t, err)
		assert.JSONEq(t, tc.want, string(body))
	}
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
