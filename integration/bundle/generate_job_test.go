package bundle_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGenerateFromExistingJobAndDeploy(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generateJobTest{T: wt, w: wt.W}

	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})

	jobId := gt.createTestJob(ctx)
	t.Cleanup(func() {
		gt.destroyJob(ctx, jobId)
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "generate", "job",
		"--existing-job-id", strconv.FormatInt(jobId, 10),
		"--config-dir", filepath.Join(bundleRoot, "resources"),
		"--source-dir", filepath.Join(bundleRoot, "src"))
	_, _, err := c.Run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "test.py"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(bundleRoot, "resources", "generated_job_*.yml"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// check the content of generated yaml
	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	generatedYaml := string(data)
	require.Contains(t, generatedYaml, "notebook_task:")
	require.Contains(t, generatedYaml, "notebook_path: "+filepath.Join("..", "src", "test.py"))
	require.Contains(t, generatedYaml, "task_key: test")
	require.Contains(t, generatedYaml, "new_cluster:")
	require.Contains(t, generatedYaml, "spark_version: 13.3.x-scala2.12")
	require.Contains(t, generatedYaml, "num_workers: 1")

	deployBundle(t, ctx, bundleRoot)

	destroyBundle(t, ctx, bundleRoot)
}

type generateJobTest struct {
	T *acc.WorkspaceT
	w *databricks.WorkspaceClient
}

func (gt *generateJobTest) createTestJob(ctx context.Context) int64 {
	t := gt.T
	w := gt.w

	tmpdir := acc.TemporaryWorkspaceDir(t, "generate-job-")
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	err = f.Write(ctx, "test.py", strings.NewReader("# Databricks notebook source\nprint('Hello world!'))"))
	require.NoError(t, err)

	resp, err := w.Jobs.Create(ctx, jobs.CreateJob{
		Name: testutil.RandomName("generated-job-"),
		Tasks: []jobs.Task{
			{
				TaskKey: "test",
				NewCluster: &compute.ClusterSpec{
					SparkVersion: "13.3.x-scala2.12",
					NumWorkers:   1,
					NodeTypeId:   testutil.GetCloud(t).NodeTypeID(),
					SparkConf: map[string]string{
						"spark.databricks.enableWsfs":                         "true",
						"spark.databricks.hive.metastore.glueCatalog.enabled": "true",
						"spark.databricks.pip.ignoreSSL":                      "true",
					},
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

func (gt *generateJobTest) destroyJob(ctx context.Context, jobId int64) {
	err := gt.w.Jobs.Delete(ctx, jobs.DeleteJob{
		JobId: jobId,
	})
	require.NoError(gt.T, err)
}
