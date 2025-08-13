// Only build this package if the dbr tag is set.
// We do not want to run this test on normal integration test runs.

// TODO: enable this tag once the development for the test is complete.
// go:build dbr

package acceptance_test

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupTest(ctx context.Context, t *testing.T, uniqueId string) (*databricks.WorkspaceClient, filer.Filer, string) {
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
	RunCommand(t, []string{"go", "run", ".", "_build/cli.tar.gz"}, "../internal/testarchive")

	archiveReader, err := os.Open("../internal/testarchive/_build/cli.tar.gz")
	require.NoError(t, err)

	t.Logf("Uploading archive...")

	err = f.Write(ctx, "cli.tar.gz", archiveReader)
	require.NoError(t, err)

	err = archiveReader.Close()
	require.NoError(t, err)

	return "cli.tar.gz"
}

func uploadRunner(ctx context.Context, t *testing.T, f filer.Filer, testDir string) string {
	runnerReader, err := os.Open("./dbr_runner.py")
	require.NoError(t, err)

	t.Logf("Uploading DBR runner...")

	// TODO: Remove the overwrite if exists flag from here.
	err = f.Write(ctx, "dbr_runner.py", runnerReader, filer.OverwriteIfExists)
	require.NoError(t, err)

	err = runnerReader.Close()
	require.NoError(t, err)

	return "dbr_runner"
}

func runDbrTests(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, runnerPath string, archivePath string) {
	t.Logf("Submitting test runner job...")
	job, err := w.Jobs.Submit(ctx, jobs.SubmitRun{
		Tasks: []jobs.SubmitTask{
			{
				TaskKey: "dbr_runner",
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: runnerPath,
					BaseParameters: map[string]string{
						"cli_archive": archivePath,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	t.Logf("Waiting for test runner job to finish...")
	run, err := job.Get()
	require.NoError(t, err)

	t.Logf("The test runner job finished with status: %s. Run URL: %s", run.State.LifeCycleState, run.RunPageUrl)
}

func TestDbrAcceptance(t *testing.T) {
	ctx := context.Background()
	uniqueId := uuid.New().String()

	w, f, testDir := setupTest(ctx, t, uniqueId)

	t.Logf("Test directory for the DBR runner: %s", testDir)

	// We compile and upload an archive of the entire repo to the workspace.
	// Only files tracked by git and binaries required by acceptance tests like
	// go, uv, jq, etc. are included.
	archiveName := buildAndUploadArchive(ctx, t, f)
	runnerName := uploadRunner(ctx, t, f, testDir)

	runDbrTests(ctx, t, w, path.Join(testDir, runnerName), path.Join(testDir, archiveName))
}
