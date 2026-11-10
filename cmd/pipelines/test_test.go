package pipelines

import (
	"errors"
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
