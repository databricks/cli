package acceptance_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testarchive"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workspaceTmpDir creates a temporary directory in the workspace for running tests.
// This is used by acceptance tests when running with the -workspace-tmp-dir flag.
func workspaceTmpDir(ctx context.Context, t *testing.T) string {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	timestamp := time.Now().Format("2006-01-02T15:04:05Z")
	path := fmt.Sprintf(
		"/Workspace/Users/%s/acceptance/%s/%s",
		currentUser.UserName,
		timestamp,
		uuid.New().String(),
	)

	t.Cleanup(func() {
		// Use FUSE for cleanup to ensure proper operation ordering.
		// Mixing FUSE (for writes) with API (for delete) can cause
		// AsyncFlushFailedException because FUSE may have pending
		// async writes that try to flush after API has deleted the directory.
		err := os.RemoveAll(path)
		assert.NoError(t, err)
	})

	// Create the directory using FUSE directly via os.MkdirAll.
	// This ensures the directory is immediately visible through the FUSE mount.
	// Using the SDK's MkdirsByPath can cause eventual consistency issues where
	// FUSE doesn't see the directory immediately after creation.
	err = os.MkdirAll(path, 0o755)
	require.NoError(t, err, "Failed to create directory %s via FUSE", path)

	// Return the FUSE path for local file operations.
	return path
}

// dbrTestConfig holds the configuration for a DBR test run.
type dbrTestConfig struct {
	// cloudTestFilter is a regex filter for cloud acceptance tests (Cloud=true).
	// These tests run with CLOUD_ENV set and workspace access.
	// If empty, all cloud tests are run.
	cloudTestFilter string

	// timeout is the maximum duration to wait for the job to complete.
	timeout time.Duration

	// verbose enables detailed output during test setup.
	// If false, only essential information is printed.
	verbose bool
}

// setupDbrTestDir creates the test directory and returns the workspace client and filer.
// It returns the API path (without /Workspace prefix) for use with workspace APIs.
func setupDbrTestDir(ctx context.Context, t *testing.T, uniqueID string) (*databricks.WorkspaceClient, filer.Filer, string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	// API path (without /Workspace prefix) for workspace API calls.
	apiPath := path.Join("/Users", currentUser.UserName, "dbr-acceptance-test", uniqueID)

	err = w.Workspace.MkdirsByPath(ctx, apiPath)
	require.NoError(t, err)

	// Note: We do not cleanup test directories created here. They are kept around
	// to enable debugging of failures or analyzing the logs.
	// They will automatically be cleaned by the nightly cleanup scripts.

	f, err := filer.NewWorkspaceFilesClient(w, apiPath)
	require.NoError(t, err)

	return w, f, apiPath
}

// buildAndUploadArchive builds the test archive and uploads it to the workspace.
func buildAndUploadArchive(ctx context.Context, t *testing.T, f filer.Filer, verbose bool) string {
	// Control testarchive verbosity
	testarchive.Verbose = verbose

	// Create temporary directories for the archive
	archiveDir := t.TempDir()
	binDir := t.TempDir()

	// Get the repo root (parent of acceptance directory)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Join(cwd, "..")

	if verbose {
		t.Log("Building test archive...")
	}
	err = testarchive.CreateArchive(archiveDir, binDir, repoRoot)
	require.NoError(t, err)

	archivePath := filepath.Join(archiveDir, "archive.tar.gz")
	archiveReader, err := os.Open(archivePath)
	require.NoError(t, err)
	defer archiveReader.Close()

	if verbose {
		t.Log("Uploading archive to workspace...")
	}
	err = f.Write(ctx, "archive.tar.gz", archiveReader)
	require.NoError(t, err)

	return "archive.tar.gz"
}

// uploadRunner uploads the DBR runner notebook to the workspace using workspace.Import.
func uploadRunner(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, testDir string, verbose bool) string {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	runnerPath := filepath.Join(cwd, "dbr_runner.py")
	runnerContent, err := os.ReadFile(runnerPath)
	require.NoError(t, err)

	notebookPath := path.Join(testDir, "dbr_runner")

	if verbose {
		t.Log("Uploading DBR runner notebook...")
	}
	err = w.Workspace.Import(ctx, workspace.Import{
		Path:      notebookPath,
		Overwrite: true,
		Language:  workspace.LanguagePython,
		Format:    workspace.ImportFormatSource,
		Content:   base64.StdEncoding.EncodeToString(runnerContent),
	})
	require.NoError(t, err)

	return "dbr_runner"
}

// buildBaseParams builds the common parameters for test tasks.
func buildBaseParams(testDir, archiveName, debugLogPath string) map[string]string {
	return map[string]string{
		"archive_path":              path.Join(testDir, archiveName),
		"cloud_env":                 os.Getenv("CLOUD_ENV"),
		"test_default_warehouse_id": os.Getenv("TEST_DEFAULT_WAREHOUSE_ID"),
		"test_default_cluster_id":   os.Getenv("TEST_DEFAULT_CLUSTER_ID"),
		"test_instance_pool_id":     os.Getenv("TEST_INSTANCE_POOL_ID"),
		"test_metastore_id":         os.Getenv("TEST_METASTORE_ID"),
		"test_user_email":           os.Getenv("TEST_USER_EMAIL"),
		"test_sp_application_id":    os.Getenv("TEST_SP_APPLICATION_ID"),
		"debug_log_path":            debugLogPath,
	}
}

// runDbrTests creates a job and runs it to execute cloud and local acceptance tests on DBR.
func runDbrTests(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, testDir, archiveName, runnerName string, config dbrTestConfig) {
	cloudEnv := os.Getenv("CLOUD_ENV")
	if cloudEnv == "" {
		t.Fatal("CLOUD_ENV is not set. Please run DBR tests from a CI environment with deco env run.")
	}

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	// Create debug logs directory
	debugLogsDir := path.Join("/Users", currentUser.UserName, "dbr_acceptance_tests")
	err = w.Workspace.MkdirsByPath(ctx, debugLogsDir)
	require.NoError(t, err)

	// Create an empty debug log file so we can get its URL before the job runs.
	// This allows us to provide the URL upfront for users to follow along.
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	debugLogFileName := fmt.Sprintf("debug-cloud-%s-%s.log", timestamp, uuid.New().String()[:8])
	debugLogPath := path.Join(debugLogsDir, debugLogFileName)

	// Create empty file via workspace API
	err = w.Workspace.Import(ctx, workspace.Import{
		Path:      debugLogPath,
		Overwrite: true,
		Format:    workspace.ImportFormatAuto,
		Content:   base64.StdEncoding.EncodeToString([]byte("")),
	})
	require.NoError(t, err)

	// Get the file's object ID for the URL
	debugLogStatus, err := w.Workspace.GetStatusByPath(ctx, debugLogPath)
	require.NoError(t, err)

	// Build cloud test parameters (Cloud=true tests, run with CLOUD_ENV set)
	cloudParams := buildBaseParams(testDir, archiveName, debugLogPath)
	cloudParams["test_type"] = "cloud"
	cloudParams["test_filter"] = config.cloudTestFilter

	jobName := "DBR Tests"
	if config.cloudTestFilter != "" {
		jobName = fmt.Sprintf("DBR Tests (%s)", config.cloudTestFilter)
	}

	// Print summary of what will run
	t.Log("")
	t.Log("=== DBR Test Run ===")
	if config.cloudTestFilter != "" {
		t.Logf("  Cloud tests: %s", config.cloudTestFilter)
	} else {
		t.Log("  Cloud tests: (all)")
	}

	notebookPath := path.Join(testDir, runnerName)

	// Create a job (not a one-time run) so we can use MaxRetries on tasks.
	// Always use serverless compute.
	t.Log("  Cluster: serverless")
	createJob := jobs.CreateJob{
		Name: jobName,
		Environments: []jobs.JobEnvironment{
			{
				EnvironmentKey: "default",
				Spec: &compute.Environment{
					EnvironmentVersion: "4",
				},
			},
		},
		Tasks: []jobs.Task{
			{
				TaskKey:        "cloud_tests",
				EnvironmentKey: "default",
				MaxRetries:     0,
				NotebookTask: &jobs.NotebookTask{
					NotebookPath:   notebookPath,
					BaseParameters: cloudParams,
					Source:         jobs.SourceWorkspace,
				},
			},
		},
	}

	// Create the job
	job, err := w.Jobs.Create(ctx, createJob)
	require.NoError(t, err)

	// The job is not deleted after the test completes.
	// This is to enable debugging of failures or analyzing the logs.
	// It will automatically be cleaned by the nightly cleanup scripts.

	// Trigger a run of the job
	wait, err := w.Jobs.RunNow(ctx, jobs.RunNow{JobId: job.JobId})
	require.NoError(t, err)

	// Fetch run details immediately to get the URL so users can follow along
	runDetails, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: wait.RunId})
	require.NoError(t, err)

	t.Log("")
	t.Log("=== Job Started ===")
	t.Logf("  Run URL: %s", runDetails.RunPageUrl)
	t.Logf("  Debug logs: %s/editor/files/%d", w.Config.Host, debugLogStatus.ObjectId)
	t.Log("")
	t.Logf("Waiting for completion (timeout: %v)...", config.timeout)

	run, err := wait.GetWithTimeout(config.timeout)
	if err != nil {
		// Try to fetch the run details for the URL and task output
		runDetails, fetchErr := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: wait.RunId})
		if fetchErr == nil {
			// Try to get the task output for debugging
			for _, task := range runDetails.Tasks {
				output, outputErr := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{
					RunId: task.RunId,
				})
				if outputErr == nil {
					if output.Error != "" {
						t.Logf("Task %s error: %s", task.TaskKey, output.Error)
					}
					if output.ErrorTrace != "" {
						t.Logf("Task %s error trace:\n%s", task.TaskKey, output.ErrorTrace)
					}
				}
			}
		}
		require.NoError(t, err)
	}

	t.Logf("Job completed. Status: %s", run.State.ResultState)

	// Check if the job succeeded
	if run.State.ResultState != jobs.RunResultStateSuccess {
		// Try to get the task output for debugging
		for _, task := range run.Tasks {
			output, outputErr := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{
				RunId: task.RunId,
			})
			if outputErr == nil && output.Error != "" {
				t.Logf("Task %s error: %s", task.TaskKey, output.Error)
			}
			if outputErr == nil && output.ErrorTrace != "" {
				t.Logf("Task %s error trace:\n%s", task.TaskKey, output.ErrorTrace)
			}
		}
		t.Fatalf("Job failed with state: %s. Check the run URL for details: %s", run.State.ResultState, run.RunPageUrl)
	}

	t.Log("All tests passed!")
}

// runDbrAcceptanceTests is the main entry point for running DBR acceptance tests.
func runDbrAcceptanceTests(t *testing.T, config dbrTestConfig) {
	ctx := context.Background()
	uniqueID := uuid.New().String()

	w, f, testDir := setupDbrTestDir(ctx, t, uniqueID)
	if config.verbose {
		t.Logf("Test directory: %s", testDir)
	}

	archiveName := buildAndUploadArchive(ctx, t, f, config.verbose)
	runnerName := uploadRunner(ctx, t, w, testDir, config.verbose)

	runDbrTests(ctx, t, w, testDir, archiveName, runnerName, config)
}

// TestDbrAcceptance runs all acceptance and integration tests on DBR using serverless compute.
// Only acceptance tests with RunsOnDbr=true in their test.toml will be executed.
// Both test types run in parallel tasks.
//
// Run with:
//
//	deco env run -i -n aws-prod-ucws -- go test -v -timeout 4h -run TestDbrAcceptance$ ./acceptance
//	OR
//	make dbr-test
func TestDbrAcceptance(t *testing.T) {
	if os.Getenv("DBR_ENABLED") != "true" {
		t.Skip("Skipping DBR test: DBR_ENABLED not set")
	}

	if os.Getenv("CLOUD_ENV") == "" {
		t.Skip("Skipping DBR test: CLOUD_ENV not set")
	}

	runDbrAcceptanceTests(t, dbrTestConfig{
		timeout: 3 * time.Hour,
		verbose: os.Getenv("DBR_TEST_VERBOSE") != "",
	})
}
