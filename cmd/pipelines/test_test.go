package pipelines

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
