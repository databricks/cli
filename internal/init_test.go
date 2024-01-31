package internal

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleInitErrorOnUnknownFields(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	_, _, err := RequireErrorRun(t, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, "failed to compute file content for bar.tmpl. variable \"does_not_exist\" not defined")
}

func TestAccBundleInitShortNameHelper(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	RequireSuccessfulRun(t, "bundle", "init", "./testdata/init/helper-short-name", "--output-dir", tmpDir)

	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)

	// Assert that short name was correctly computed.
	assertLocalFileContents(t, filepath.Join(tmpDir, "foo.txt"), auth.GetShortUserName(me.UserName))
}

func TestAccBundleInitUserNameHelper(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	RequireSuccessfulRun(t, "bundle", "init", "./testdata/init/helper-user-name", "--output-dir", tmpDir)

	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)

	// Assert that user name was correctly computed.
	assertLocalFileContents(t, filepath.Join(tmpDir, "foo.txt"), me.UserName)
}

func TestAccBundleInitWorkspaceHostHelper(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "DATABRICKS_HOST"))

	tmpDir := t.TempDir()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	RequireSuccessfulRun(t, "bundle", "init", "./testdata/init/helper-workspace-host", "--output-dir", tmpDir)

	// Assert that workspace host was correctly computed.
	assertLocalFileContents(t, filepath.Join(tmpDir, "foo.txt"), w.Config.Host)
}
