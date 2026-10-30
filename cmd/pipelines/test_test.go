package pipelines

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

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

// TestValidatePytestArgsRejectsUnsupportedOptions covers every pytest option that the API rejects.
func TestValidatePytestArgsRejectsUnsupportedOptions(t *testing.T) {
	for _, arg := range []string{
		"--collect-only", "-x", "--exitfirst", "--maxfail=1", "--lf", "--ff", "--cache-clear", "-n",
		"--pdb", "--trace", "-q", "-v", "--color=yes", "--tb=short", "-s", "--capture=no", "-p",
		"--junitxml=out.xml", "--html=out.html", "--basetemp=/tmp", "-c", "--rootdir=/", "--confcutdir=/",
		"--pyargs", "--timeout=10", "--no-wait", "--junit-xml=out.xml",
	} {
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

func TestBundleTestSelectionPath(t *testing.T) {
	syncRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(syncRoot, "pipeline", "tests"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(syncRoot, "pipeline", "tests", "test_a.py"), nil, 0o644))
	filePath := "/Workspace/Users/someone/.bundle/orders/dev/files"
	base := func() (string, error) { return filePath + "/pipeline", nil }
	noBase := func() (string, error) { return "", nil }

	// A path that exists where the user stands is a local path.
	got, err := bundleTestSelectionPath(filepath.Join(syncRoot, "pipeline"), syncRoot, filePath, base, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filePath+"/pipeline/tests/test_a.py", got)

	// Otherwise it is a test ID, relative to the folder that test IDs are relative to.
	got, err = bundleTestSelectionPath(syncRoot, syncRoot, filePath, base, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filePath+"/pipeline/tests/test_a.py", got)

	// Without that folder, it is a local path.
	got, err = bundleTestSelectionPath(syncRoot, syncRoot, filePath, noBase, "tests/test_a.py")
	require.NoError(t, err)
	assert.Equal(t, filePath+"/tests/test_a.py", got)

	// A path that leaves that folder is a local path too.
	for _, testIDBase := range []func() (string, error){base, noBase} {
		_, err = bundleTestSelectionPath(syncRoot, syncRoot, filePath, testIDBase, filepath.Join("..", "tests"))
		assert.ErrorContains(t, err, "is outside the synchronized bundle files in "+syncRoot)
	}
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
