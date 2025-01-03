package bundle_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGenerateFromExistingPipelineAndDeploy(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generatePipelineTest{T: wt, w: wt.W}

	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})

	pipelineId, name := gt.createTestPipeline(ctx)
	t.Cleanup(func() {
		gt.destroyPipeline(ctx, pipelineId)
	})

	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "generate", "pipeline",
		"--existing-pipeline-id", pipelineId,
		"--config-dir", filepath.Join(bundleRoot, "resources"),
		"--source-dir", filepath.Join(bundleRoot, "src"))
	_, _, err := c.Run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "notebook.py"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "test.py"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(bundleRoot, "resources", "generated_pipeline_*.yml"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// check the content of generated yaml
	fileName := matches[0]
	data, err := os.ReadFile(fileName)
	require.NoError(t, err)
	generatedYaml := string(data)

	// Replace pipeline name
	generatedYaml = strings.ReplaceAll(generatedYaml, name, testutil.RandomName("copy-generated-pipeline-"))
	err = os.WriteFile(fileName, []byte(generatedYaml), 0o644)
	require.NoError(t, err)

	require.Contains(t, generatedYaml, "libraries:")
	require.Contains(t, generatedYaml, "- notebook:")
	require.Contains(t, generatedYaml, "path: "+filepath.Join("..", "src", "notebook.py"))
	require.Contains(t, generatedYaml, "- file:")
	require.Contains(t, generatedYaml, "path: "+filepath.Join("..", "src", "test.py"))

	deployBundle(t, ctx, bundleRoot)

	destroyBundle(t, ctx, bundleRoot)
}

type generatePipelineTest struct {
	T *acc.WorkspaceT
	w *databricks.WorkspaceClient
}

func (gt *generatePipelineTest) createTestPipeline(ctx context.Context) (string, string) {
	t := gt.T
	w := gt.w

	tmpdir := acc.TemporaryWorkspaceDir(t, "generate-pipeline-")
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	err = f.Write(ctx, "notebook.py", strings.NewReader("# Databricks notebook source\nprint('Hello world!'))"))
	require.NoError(t, err)

	err = f.Write(ctx, "test.py", strings.NewReader("print('Hello!')"))
	require.NoError(t, err)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()

	name := testutil.RandomName("generated-pipeline-")
	resp, err := w.Pipelines.Create(ctx, pipelines.CreatePipeline{
		Name: name,
		Libraries: []pipelines.PipelineLibrary{
			{
				Notebook: &pipelines.NotebookLibrary{
					Path: path.Join(tmpdir, "notebook"),
				},
			},
			{
				File: &pipelines.FileLibrary{
					Path: path.Join(tmpdir, "test.py"),
				},
			},
		},
		Clusters: []pipelines.PipelineCluster{
			{
				CustomTags: map[string]string{
					"Tag1": "Yes",
					"Tag2": "24X7",
					"Tag3": "APP-1234",
				},
				NodeTypeId: nodeTypeId,
				NumWorkers: 2,
				SparkConf: map[string]string{
					"spark.databricks.enableWsfs":                         "true",
					"spark.databricks.hive.metastore.glueCatalog.enabled": "true",
					"spark.databricks.pip.ignoreSSL":                      "true",
				},
			},
		},
	})
	require.NoError(t, err)

	return resp.PipelineId, name
}

func (gt *generatePipelineTest) destroyPipeline(ctx context.Context, pipelineId string) {
	err := gt.w.Pipelines.Delete(ctx, pipelines.DeletePipelineRequest{
		PipelineId: pipelineId,
	})
	require.NoError(gt.T, err)
}
