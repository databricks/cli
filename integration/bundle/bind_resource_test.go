package bundle_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

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
