package pipelines

import (
	"encoding/json"
	"errors"
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

func TestRootPathWorkspacePath(t *testing.T) {
	got, err := rootPathWorkspacePath("/Workspace/Users/someone/orders", "tests/test_orders.py")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/someone/orders/tests/test_orders.py", got)

	_, err = rootPathWorkspacePath("", "tests")
	assert.EqualError(t, err, "the pipeline has no root_path; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
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
