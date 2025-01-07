package bundle_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindJobToExistingJob(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: wt, w: wt.W}

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		gt.destroyJob(ctx, jobId)
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "deployment", "bind", "foo", strconv.FormatInt(jobId, 10), "--auto-approve")
	_, _, err := c.Run()
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	deployBundle(t, ctx, bundleRoot)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// Check that job is bound and updated with config from bundle
	job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)
	require.Equal(t, job.Settings.Name, "test-job-basic-"+uniqueId)
	require.Contains(t, job.Settings.Tasks[0].SparkPythonTask.PythonFile, "hello_world.py")

	c = testcli.NewRunner(t, ctx, "bundle", "deployment", "unbind", "foo")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	destroyBundle(t, ctx, bundleRoot)

	// Check that job is unbound and exists after bundle is destroyed
	job, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)
	require.Equal(t, job.Settings.Name, "test-job-basic-"+uniqueId)
	require.Contains(t, job.Settings.Tasks[0].SparkPythonTask.PythonFile, "hello_world.py")
}

func TestAbortBind(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: wt, w: wt.W}

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		gt.destroyJob(ctx, jobId)
		destroyBundle(t, ctx, bundleRoot)
	})

	// Bind should fail because prompting is not possible.
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	c := testcli.NewRunner(t, ctx, "bundle", "deployment", "bind", "foo", strconv.FormatInt(jobId, 10))

	// Expect error suggesting to use --auto-approve
	_, _, err := c.Run()
	assert.ErrorContains(t, err, "failed to bind the resource")
	assert.ErrorContains(t, err, "This bind operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")

	deployBundle(t, ctx, bundleRoot)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// Check that job is not bound and not updated with config from bundle
	job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)

	require.NotEqual(t, job.Settings.Name, "test-job-basic-"+uniqueId)
	require.Contains(t, job.Settings.Tasks[0].NotebookTask.NotebookPath, "test")
}

func TestGenerateAndBind(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: wt, w: wt.W}

	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
			JobId: jobId,
		})
		if err == nil {
			gt.destroyJob(ctx, jobId)
		}
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "generate", "job",
		"--key", "test_job_key",
		"--existing-job-id", strconv.FormatInt(jobId, 10),
		"--config-dir", filepath.Join(bundleRoot, "resources"),
		"--source-dir", filepath.Join(bundleRoot, "src"))
	_, _, err = c.Run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "test.py"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(bundleRoot, "resources", "test_job_key.job.yml"))
	require.NoError(t, err)

	require.Len(t, matches, 1)

	c = testcli.NewRunner(t, ctx, "bundle", "deployment", "bind", "test_job_key", strconv.FormatInt(jobId, 10), "--auto-approve")
	_, _, err = c.Run()
	require.NoError(t, err)

	deployBundle(t, ctx, bundleRoot)

	destroyBundle(t, ctx, bundleRoot)

	// Check that job is bound and does not extsts after bundle is destroyed
	_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.ErrorContains(t, err, "does not exist.")
}
