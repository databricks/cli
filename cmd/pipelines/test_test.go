package pipelines

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"

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
		{args: []string{"-msmoke,fast"}, wantErr: `pytest argument "-msmoke,fast" must be split: pass --pytest-arg=-m --pytest-arg='smoke,fast'`},
		{args: []string{"-k=orders and not slow"}, wantErr: `pytest argument "-k=orders and not slow" must be split: pass --pytest-arg=-k --pytest-arg='orders and not slow'`},
		{args: []string{"-k", " "}, wantErr: "pytest argument -k requires a non-empty value"},
		{args: []string{"-k", "-m"}, wantErr: `pytest argument -k requires a value, not "-m": pass it in the next --pytest-arg`},
		{args: []string{"-m", "--maxfail=1"}, wantErr: `pytest argument -m requires a value, not "--maxfail=1": pass it in the next --pytest-arg`},
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

func TestRootPathSelection(t *testing.T) {
	got, err := rootPathSelection("/Workspace/Users/someone/orders", "tests/test_orders.py")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/someone/orders/tests/test_orders.py", got)

	_, err = rootPathSelection("", "tests")
	assert.EqualError(t, err, "the pipeline has no root_path; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
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

	// A local path maps to its deployed path, relative to where the user stands.
	got, err := bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline"), syncRoot, filesRoot, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/pipeline/tests/test_a.py", got)

	got, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, "pipeline/tests")
	require.NoError(t, err)
	assert.Equal(t, filesRoot+"/pipeline/tests", got)

	// A path that does not exist locally is an error, so that a typo fails before compute starts.
	_, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, "tests/test_a.py")
	assert.EqualError(t, err, filepath.ToSlash(filepath.Join(syncRoot, "tests", "test_a.py"))+" does not exist; pass a local path or an absolute workspace path")

	// A path outside the synchronized files is an error. The error shows forward slashes.
	_, err = bundleTestSelectionPath(syncRoot, syncRoot, filesRoot, filepath.Join("..", "tests"))
	assert.ErrorContains(t, err, "is outside the synchronized bundle files in "+filepath.ToSlash(syncRoot))

	// When the deployed path of local files is unknown, a local path is an error.
	_, err = bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline"), syncRoot, "", "tests/test_a.py")
	assert.EqualError(t, err, "local test paths are not supported with workspace.sync_mode immutable; pass an absolute workspace path")
}

func TestHasRelativeSelection(t *testing.T) {
	assert.False(t, hasRelativeSelection(nil))
	assert.False(t, hasRelativeSelection([]string{"/Workspace/tests", "/Workspace/a.py::test_x"}))
	assert.True(t, hasRelativeSelection([]string{"/Workspace/tests", "tests/test_a.py::test_x"}))
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
