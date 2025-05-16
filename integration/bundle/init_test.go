package bundle_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleInitErrorOnUnknownFields(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	_, _, err := testcli.RequireErrorRun(t, ctx, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, "failed to compute file content for bar.tmpl. variable \"does_not_exist\" not defined")
}

func TestBundleInitHelpers(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	var smallestNode string
	switch testutil.GetCloud(t) {
	case testutil.Azure:
		smallestNode = "Standard_D3_v2"
	case testutil.GCP:
		smallestNode = "n1-standard-4"
	case testutil.AWS:
		smallestNode = "i3.xlarge"
	default:
		t.Fatal("Unknown cloud environment")
	}

	tests := []struct {
		funcName string
		expected string
	}{
		{
			funcName: "{{short_name}}",
			expected: iamutil.GetShortUserName(me),
		},
		{
			funcName: "{{user_name}}",
			expected: me.UserName,
		},
		{
			funcName: "{{workspace_host}}",
			expected: w.Config.Host,
		},
		{
			funcName: "{{is_service_principal}}",
			expected: strconv.FormatBool(iamutil.IsServicePrincipal(me)),
		},
		{
			funcName: "{{smallest_node_type}}",
			expected: smallestNode,
		},
	}

	for _, test := range tests {
		// Setup template to test the helper function.
		tmpDir := t.TempDir()
		tmpDir2 := t.TempDir()

		err := os.Mkdir(filepath.Join(tmpDir, "template"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "template", "foo.txt.tmpl"), []byte(test.funcName), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "databricks_template_schema.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		// Run bundle init.
		testcli.RequireSuccessfulRun(t, ctx, "bundle", "init", tmpDir, "--output-dir", tmpDir2)

		// Assert that the helper function was correctly computed.
		contents := testutil.ReadFile(t, filepath.Join(tmpDir2, "foo.txt"))
		assert.Contains(t, contents, test.expected)
	}
}
