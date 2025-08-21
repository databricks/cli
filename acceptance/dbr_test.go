//go:build dbr_only

package acceptance_test

import (
	"context"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupTestDir(ctx context.Context, t *testing.T, uniqueId string) (*databricks.WorkspaceClient, filer.Filer, string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	testDir := path.Join("/Workspace/Users/", currentUser.UserName, "acceptance", uniqueId)

	err = w.Workspace.MkdirsByPath(ctx, testDir)
	require.NoError(t, err)

	f, err := filer.NewWorkspaceFilesClient(w, testDir)
	require.NoError(t, err)

	return w, f, testDir
}

func buildAndUploadArchive(ctx context.Context, t *testing.T, f filer.Filer) string {
	// Build the CLI archives and upload to the workspace.
	RunCommand(t, []string{"go", "run", "."}, "../internal/testarchive")

	archiveReader, err := os.Open("../internal/testarchive/_build/archive.tar.gz")
	require.NoError(t, err)

	t.Logf("Uploading archive...")
	err = f.Write(ctx, "archive.tar.gz", archiveReader)
	require.NoError(t, err)

	err = archiveReader.Close()
	require.NoError(t, err)

	return "archive.tar.gz"
}

func uploadRunner(ctx context.Context, t *testing.T, f filer.Filer) string {
	runnerReader, err := os.Open("dbr_runner.py")
	require.NoError(t, err)

	t.Logf("Uploading DBR runner...")
	err = f.Write(ctx, "dbr_runner.py", runnerReader)
	require.NoError(t, err)

	err = runnerReader.Close()
	require.NoError(t, err)

	return "dbr_runner"
}

func runDbrTests(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, runnerPath, archivePath string) {
	t.Logf("Submitting test runner job...")

	envvars := []string{
		"CLOUD_ENV",
		"TEST_DEFAULT_CLUSTER_ID",
		"TEST_DEFAULT_WAREHOUSE_ID",
		"TEST_INSTANCE_POOL_ID",
		"TEST_METASTORE_ID",
	}

	baseParams := map[string]string{
		"archive_path": archivePath,
		"short":        strconv.FormatBool(testing.Short()),
	}
	for _, envvar := range envvars {
		baseParams[envvar] = os.Getenv(envvar)
	}

	job, err := w.Jobs.Submit(ctx, jobs.SubmitRun{
		RunName: "DBR Acceptance Tests",
		Tasks: []jobs.SubmitTask{
			{
				TaskKey: "dbr_runner",
				NotebookTask: &jobs.NotebookTask{
					NotebookPath:   runnerPath,
					BaseParameters: baseParams,
				},
			},
		},
	})
	require.NoError(t, err)

	t.Logf("Waiting for test runner job to finish...")
	run, err := job.GetWithTimeout(2 * time.Hour)
	require.NoError(t, err)

	t.Logf("The test runner job finished with status: %s. Run URL: %s", run.State.LifeCycleState, run.RunPageUrl)
}

func TestDbrAcceptance(t *testing.T) {
	ctx := context.Background()
	uniqueId := uuid.New().String()

	w, f, testDir := setupTestDir(ctx, t, uniqueId)
	t.Logf("Test directory for the DBR runner: %s", testDir)

	// We compile and upload an archive of the entire repo to the workspace.
	// Only files tracked by git and binaries required by acceptance tests like
	// go, uv, jq, etc. are included.
	archiveName := buildAndUploadArchive(ctx, t, f)
	runnerName := uploadRunner(ctx, t, f)

	runDbrTests(ctx, t, w, path.Join(testDir, runnerName), path.Join(testDir, archiveName))
}
