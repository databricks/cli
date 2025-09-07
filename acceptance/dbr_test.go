package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testarchive"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workspaceTmpDir(ctx context.Context, t *testing.T) (*databricks.WorkspaceClient, filer.Filer, string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	timestamp := time.Now().Format("2006-01-02T15:04:05Z")
	tmpDir := fmt.Sprintf(
		"/Workspace/Users/%s/acceptance/%s/%s",
		currentUser.UserName,
		timestamp,
		uuid.New().String(),
	)

	t.Cleanup(func() {
		err := w.Workspace.Delete(ctx, workspace.Delete{
			Path:      tmpDir,
			Recursive: true,
		})
		assert.NoError(t, err)
	})

	err = w.Workspace.MkdirsByPath(ctx, tmpDir)
	require.NoError(t, err)

	f, err := filer.NewWorkspaceFilesClient(w, tmpDir)
	require.NoError(t, err)

	return w, f, tmpDir
}

// Stable scratch directory to run and iterate on DBR tests.
func workspaceStableDir(ctx context.Context, t *testing.T) (w *databricks.WorkspaceClient, f filer.Filer, path string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	path = fmt.Sprintf("/Workspace/Users/%s/dbr_scratch", currentUser.UserName)

	// Delete the directory if it exists.
	err = w.Workspace.Delete(ctx, workspace.Delete{
		Path:      path,
		Recursive: true,
	})
	var aerr *apierr.APIError
	if err != nil && (!errors.As(err, &aerr) || aerr.ErrorCode != "RESOURCE_DOES_NOT_EXIST") {
		t.Fatalf("Failed to delete directory %s: %v", path, err)
	}

	err = w.Workspace.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	// Create a filer client for the workspace.
	f, err = filer.NewWorkspaceFilesClient(w, path)
	require.NoError(t, err)

	return w, f, path
}

func buildAndUploadArchive(ctx context.Context, t *testing.T, f filer.Filer) {
	archiveDir := t.TempDir()
	binDir := t.TempDir()
	archiveName := "archive.tar.gz"

	// Build the CLI archives and upload to the workspace.
	testarchive.CreateArchive(archiveDir, binDir, "..")

	archiveReader, err := os.Open(filepath.Join(archiveDir, archiveName))
	require.NoError(t, err)

	err = f.Write(ctx, archiveName, archiveReader)
	require.NoError(t, err)

	err = archiveReader.Close()
	require.NoError(t, err)
}

func uploadScratchRunner(ctx context.Context, t *testing.T, f filer.Filer, w *databricks.WorkspaceClient, dir string) string {
	runnerReader, err := os.Open("scratch_dbr_runner.ipynb")
	require.NoError(t, err)

	err = f.Write(ctx, "scratch_dbr_runner.ipynb", runnerReader)
	require.NoError(t, err)

	err = runnerReader.Close()
	require.NoError(t, err)

	status, err := w.Workspace.GetStatusByPath(ctx, path.Join(dir, "scratch_dbr_runner"))
	require.NoError(t, err)

	url := w.Config.Host + "/editor/notebooks/" + strconv.FormatInt(status.ObjectId, 10)

	return url
}

func uploadParams(ctx context.Context, t *testing.T, f filer.Filer) {
	names := []string{
		"CLOUD_ENV",
		"TEST_DEFAULT_CLUSTER_ID",
		"TEST_DEFAULT_WAREHOUSE_ID",
		"TEST_INSTANCE_POOL_ID",
		"TEST_METASTORE_ID",
	}

	env := make(map[string]string)
	for _, name := range names {
		env[name] = os.Getenv(name)
	}

	b, err := json.MarshalIndent(env, "", "  ")
	require.NoError(t, err)

	err = f.Write(ctx, "params.json", bytes.NewReader(b))
	require.NoError(t, err)
}

// Running this test will setup a DBR test runner the configured workspace.
// You'll need to run the tests by actually running the notebook on the workspace.
func TestSetupDbrRunner(t *testing.T) {
	ctx := t.Context()
	w, f, dir := workspaceStableDir(ctx, t)

	t.Logf("Building and uploading archive...")
	buildAndUploadArchive(ctx, t, f)

	t.Logf("Uploading params...")
	uploadParams(ctx, t, f)

	t.Logf("Uploading runner...")
	url := uploadScratchRunner(ctx, t, f, w, dir)

	t.Logf("Created DBR testing notebook at: %s", url)
}

func TestArchive(t *testing.T) {
	archiveDir := t.TempDir()
	binDir := t.TempDir()
	testarchive.CreateArchive(archiveDir, binDir, "..")

	assert.FileExists(t, filepath.Join(archiveDir, "archive.tar.gz"))
}
