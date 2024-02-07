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
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccGenerateFromExistingPipelineAndDeploy(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	gt := &generatePipelineTest{T: t, w: wt.W}

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "with_includes", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	pipelineId := gt.createTestPipeline(ctx)
	t.Cleanup(func() {
		gt.destroyPipeline(ctx, pipelineId)
	})

	t.Setenv("BUNDLE_ROOT", bundleRoot)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "generate", "pipeline",
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

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	err = destroyBundle(t, ctx, bundleRoot)
	require.NoError(t, err)
}

type generatePipelineTest struct {
	T *testing.T
	w *databricks.WorkspaceClient
}

func (gt *generatePipelineTest) createTestPipeline(ctx context.Context) string {
	t := gt.T
	w := gt.w

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

func (gt *generatePipelineTest) destroyPipeline(ctx context.Context, pipelineId string) {
	err := gt.w.Pipelines.Delete(ctx, pipelines.DeletePipelineRequest{
		PipelineId: pipelineId,
	})
	require.NoError(gt.T, err)
}
