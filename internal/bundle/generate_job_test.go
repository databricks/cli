package bundle

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccGenerateFromExistingJobAndDeploy(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	jobId := createTestJob(t)
	t.Cleanup(func() {
		destroyJob(t, jobId)
		require.NoError(t, err)
	})

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	c := internal.NewCobraTestRunner(t, "bundle", "generate", "job",
		"--existing-job-id", fmt.Sprint(jobId),
		"--config-dir", filepath.Join(bundleRoot, "resources"),
		"--source-dir", filepath.Join(bundleRoot, "src"))
	_, _, err = c.Run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "test.py"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(bundleRoot, "resources", "job_generated_job_*.yml"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// check the content of generated yaml
	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	generatedYaml := string(data)
	require.Contains(t, generatedYaml, "notebook_task:")
	require.Contains(t, generatedYaml, "notebook_path: ../src/test.py")
	require.Contains(t, generatedYaml, "task_key: test")
	require.Contains(t, generatedYaml, "new_cluster:")
	require.Contains(t, generatedYaml, "spark_version: 13.3.x-scala2.12")
	require.Contains(t, generatedYaml, "num_workers: 1")

	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	err = destroyBundle(t, bundleRoot)
	require.NoError(t, err)

}

func createTestJob(t *testing.T) int64 {
	var nodeTypeId string
	switch testutil.GetCloud(t) {
	case testutil.AWS:
		nodeTypeId = "i3.xlarge"
	case testutil.Azure:
		nodeTypeId = "Standard_DS4_v2"
	case testutil.GCP:
		nodeTypeId = "n1-standard-4"
	}

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	tmpdir := internal.TemporaryWorkspaceDir(t, w)
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	err = f.Write(ctx, "test.py", strings.NewReader("# Databricks notebook source\nprint('Hello world!'))"))
	require.NoError(t, err)

	resp, err := w.Jobs.Create(ctx, jobs.CreateJob{
		Name: internal.RandomName("generated-job-"),
		Tasks: []jobs.Task{
			{
				TaskKey: "test",
				NewCluster: &compute.ClusterSpec{
					SparkVersion: "13.3.x-scala2.12",
					NumWorkers:   1,
					NodeTypeId:   nodeTypeId,
				},
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: path.Join(tmpdir, "test"),
				},
			},
		},
	})
	require.NoError(t, err)

	return resp.JobId
}

func destroyJob(t *testing.T, jobId int64) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	err = w.Jobs.Delete(ctx, jobs.DeleteJob{
		JobId: jobId,
	})
	require.NoError(t, err)
}
