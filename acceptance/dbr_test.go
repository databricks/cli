package acceptance_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func workspaceTmpDir(ctx context.Context, t *testing.T) (*databricks.WorkspaceClient, filer.Filer, string) {
	// If the test is being run on DBR, auth is already configured
	// by the dbr_runner notebook by reading a token from the notebook context and
	// setting DATABRICKS_TOKEN and DATABRICKS_HOST environment variables.
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	// Run DBR tests on the workspace file system to mimic usage from
	// DABs in the workspace.
	timestamp := time.Now().Format("2006-01-02T15:04:05Z")
	tmpDir := fmt.Sprintf(
		"/Workspace/Users/%s/acceptance/%s/%s",
		currentUser.UserName,
		timestamp,
		uuid.New().String(),
	)

	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	err = w.Workspace.MkdirsByPath(ctx, tmpDir)
	require.NoError(t, err)

	f, err := filer.NewWorkspaceFilesClient(w, tmpDir)
	require.NoError(t, err)

	return w, f, tmpDir
}

func buildAndUploadArchive(ctx context.Context, t *testing.T, f filer.Filer) string {
	pkgDir := path.Join("..", "internal", "testarchive")

	// Build the CLI archives and upload to the workspace.
	RunCommand(t, []string{"go", "run", ".", "_build", "_bin"}, pkgDir)

	archiveReader, err := os.Open(filepath.Join(pkgDir, "_build", "archive.tar.gz"))
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

	waiter, err := w.Jobs.Submit(ctx, jobs.SubmitRun{
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

	t.Logf("Waiting for test runner job to finish. Run URL: %s", urlForRun(ctx, t, w, waiter.RunId))

	var run *jobs.Run
	deadline, ok := t.Deadline()
	if ok {
		// If -timeout is configured for the test, wait until that time for the job run results.
		run, err = waiter.GetWithTimeout(time.Until(deadline))
		require.NoError(t, err)
	} else {
		// Use the default timeout from the SDK otherwise.
		run, err = waiter.Get()
		require.NoError(t, err)
	}

	t.Logf("The test runner job finished with status: %s", run.State.LifeCycleState)
}

func urlForRun(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, runId int64) string {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runId})
	require.NoError(t, err)
	return run.RunPageUrl
}

func testDbrAcceptance(t *testing.T) {
	ctx := context.Background()

	w, f, testDir := workspaceTmpDir(ctx, t)
	t.Logf("Test directory for the DBR runner: %s", testDir)

	// We compile and upload an archive of the entire repo to the workspace.
	// Only files tracked by git and binaries required by acceptance tests like
	// go, uv, jq, etc. are included.
	archiveName := buildAndUploadArchive(ctx, t, f)
	runnerName := uploadRunner(ctx, t, f)

	runDbrTests(ctx, t, w, path.Join(testDir, runnerName), path.Join(testDir, archiveName))
}
