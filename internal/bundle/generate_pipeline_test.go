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
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccGenerateFromExistingPipelineAndDeploy(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	pipelineId := createTestPipeline(t)
	t.Cleanup(func() {
		destroyPipeline(t, pipelineId)
		require.NoError(t, err)
	})

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	c := internal.NewCobraTestRunner(t, "bundle", "generate", "pipeline",
		"--existing-pipeline-id", fmt.Sprint(pipelineId),
		"--config-dir", filepath.Join(bundleRoot, "resources"),
		"--source-dir", filepath.Join(bundleRoot, "src"))
	_, _, err = c.Run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "notebook.py"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bundleRoot, "src", "test.py"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(bundleRoot, "resources", "generated_pipeline_*.yml"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// check the content of generated yaml
	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	generatedYaml := string(data)
	require.Contains(t, generatedYaml, "libraries:")
	require.Contains(t, generatedYaml, "- notebook:")
	require.Contains(t, generatedYaml, fmt.Sprintf("path: %s", filepath.Join("..", "src", "notebook.py")))
	require.Contains(t, generatedYaml, "- file:")
	require.Contains(t, generatedYaml, fmt.Sprintf("path: %s", filepath.Join("..", "src", "test.py")))

	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	err = destroyBundle(t, bundleRoot)
	require.NoError(t, err)
}

func createTestPipeline(t *testing.T) string {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	tmpdir := internal.TemporaryWorkspaceDir(t, w)
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	err = f.Write(ctx, "notebook.py", strings.NewReader("# Databricks notebook source\nprint('Hello world!'))"))
	require.NoError(t, err)

	err = f.Write(ctx, "test.py", strings.NewReader("print('Hello!')"))
	require.NoError(t, err)

	resp, err := w.Pipelines.Create(ctx, pipelines.CreatePipeline{
		Name: internal.RandomName("generated-pipeline-"),
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
	})
	require.NoError(t, err)

	return resp.PipelineId
}

func destroyPipeline(t *testing.T, pipelineId string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	err = w.Pipelines.Delete(ctx, pipelines.DeletePipelineRequest{
		PipelineId: pipelineId,
	})
	require.NoError(t, err)
}
