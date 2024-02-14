package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccBindJobToExistingJob(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: t, w: wt.W}

	nodeTypeId := internal.GetNodeTypeId(env)
	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})
	require.NoError(t, err)

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		gt.destroyJob(ctx, jobId)
		require.NoError(t, err)
	})

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	c := internal.NewCobraTestRunner(t, "bundle", "deployment", "bind", "foo", fmt.Sprint(jobId), "--auto-approve")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// Check that job is bound and updated with config from bundle
	job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)
	require.Equal(t, job.Settings.Name, fmt.Sprintf("test-job-basic-%s", uniqueId))
	require.Contains(t, job.Settings.Tasks[0].SparkPythonTask.PythonFile, "hello_world.py")

	c = internal.NewCobraTestRunner(t, "bundle", "deployment", "unbind", "foo")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment
	err = os.RemoveAll(filepath.Join(bundleRoot, ".databricks"))
	require.NoError(t, err)

	err = destroyBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// Check that job is unbound and exists after bundle is destroyed
	job, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)
	require.Equal(t, job.Settings.Name, fmt.Sprintf("test-job-basic-%s", uniqueId))
	require.Contains(t, job.Settings.Tasks[0].SparkPythonTask.PythonFile, "hello_world.py")

}

func TestAccAbortBind(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: t, w: wt.W}

	nodeTypeId := internal.GetNodeTypeId(env)
	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"spark_version": "13.3.x-scala2.12",
		"node_type_id":  nodeTypeId,
	})
	require.NoError(t, err)

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		gt.destroyJob(ctx, jobId)
		destroyBundle(t, ctx, bundleRoot)
	})

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	c := internal.NewCobraTestRunner(t, "bundle", "deployment", "bind", "foo", fmt.Sprint(jobId))

	// Simulate user aborting the bind. This is done by not providing any input to the prompt in non-interactive mode.
	_, _, err = c.Run()
	require.ErrorContains(t, err, "failed to bind the resource")

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// Check that job is not bound and not updated with config from bundle
	job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: jobId,
	})
	require.NoError(t, err)

	require.NotEqual(t, job.Settings.Name, fmt.Sprintf("test-job-basic-%s", uniqueId))
	require.Contains(t, job.Settings.Tasks[0].NotebookTask.NotebookPath, "test")
}
