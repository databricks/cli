package generate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGeneratePipelineCommand(t *testing.T) {
	cmd := NewGeneratePipelineCommand()

	root := t.TempDir()
	b := &bundle.Bundle{
		Config: config.Root{
			Path: root,
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{PipelineId: "test-pipeline"}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "test-pipeline",
		Name:       "test-pipeline",
		Spec: &pipelines.PipelineSpec{
			Name: "test-pipeline",
			Libraries: []pipelines.PipelineLibrary{
				{Notebook: &pipelines.NotebookLibrary{
					Path: "/test/notebook",
				}},
				{File: &pipelines.FileLibrary{
					Path: "/test/file.py",
				}},
			},
		},
	}, nil)

	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/test/notebook").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeNotebook,
		Language:   workspace.LanguagePython,
		Path:       "/test/notebook",
	}, nil)

	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/test/file.py").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeFile,
		Path:       "/test/file.py",
	}, nil)

	notebookContent := io.NopCloser(bytes.NewBufferString("# Databricks notebook source\nNotebook content"))
	pyContent := io.NopCloser(bytes.NewBufferString("Py content"))
	workspaceApi.EXPECT().Download(mock.Anything, "/test/notebook", mock.Anything).Return(notebookContent, nil)
	workspaceApi.EXPECT().Download(mock.Anything, "/test/file.py", mock.Anything).Return(pyContent, nil)

	cmd.SetContext(bundle.Context(context.Background(), b))
	cmd.Flag("existing-pipeline-id").Value.Set("test-pipeline")

	configDir := filepath.Join(root, "resources")
	cmd.Flag("config-dir").Value.Set(configDir)

	srcDir := filepath.Join(root, "src")
	cmd.Flag("source-dir").Value.Set(srcDir)

	var key string
	cmd.Flags().StringVar(&key, "key", "test_pipeline", "")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "test_pipeline.yml"))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`resources:
  pipelines:
    test_pipeline:
      name: test-pipeline
      libraries:
        - notebook:
            path: %s
        - file:
            path: %s
`, filepath.Join("..", "src", "notebook.py"), filepath.Join("..", "src", "file.py")), string(data))

	data, err = os.ReadFile(filepath.Join(srcDir, "notebook.py"))
	require.NoError(t, err)
	require.Equal(t, "# Databricks notebook source\nNotebook content", string(data))

	data, err = os.ReadFile(filepath.Join(srcDir, "file.py"))
	require.NoError(t, err)
	require.Equal(t, "Py content", string(data))
}
