package generate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGeneratePipelineCommand(t *testing.T) {
	cmd := NewGeneratePipelineCommand()

	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	pipelineApi := m.GetMockPipelinesAPI()
	pipelineApi.EXPECT().Get(mock.Anything, pipelines.GetPipelineRequest{PipelineId: "test-pipeline"}).Return(&pipelines.GetPipelineResponse{
		PipelineId: "test-pipeline",
		Name:       "test-pipeline",
		Spec: &pipelines.PipelineSpec{
			Name: "test-pipeline",
			Clusters: []pipelines.PipelineCluster{
				{
					CustomTags: map[string]string{
						"Tag1": "24X7-1234",
					},
				},
				{
					SparkConf: map[string]string{
						"spark.databricks.delta.preview.enabled": "true",
					},
				},
			},
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
	require.NoError(t, cmd.Flag("existing-pipeline-id").Value.Set("test-pipeline"))

	configDir := filepath.Join(root, "resources")
	require.NoError(t, cmd.Flag("config-dir").Value.Set(configDir))

	srcDir := filepath.Join(root, "src")
	require.NoError(t, cmd.Flag("source-dir").Value.Set(srcDir))

	var key string
	cmd.Flags().StringVar(&key, "key", "test_pipeline", "")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "test_pipeline.pipeline.yml"))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`resources:
  pipelines:
    test_pipeline:
      name: test-pipeline
      clusters:
        - custom_tags:
            "Tag1": "24X7-1234"
        - spark_conf:
            "spark.databricks.delta.preview.enabled": "true"
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

func TestGenerateJobCommand(t *testing.T) {
	cmd := NewGenerateJobCommand()

	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	jobsApi := m.GetMockJobsAPI()
	jobsApi.EXPECT().Get(mock.Anything, jobs.GetJobRequest{JobId: 1234}).Return(&jobs.Job{
		Settings: &jobs.JobSettings{
			Name: "test-job",
			JobClusters: []jobs.JobCluster{
				{NewCluster: compute.ClusterSpec{
					CustomTags: map[string]string{
						"Tag1": "24X7-1234",
					},
				}},
				{NewCluster: compute.ClusterSpec{
					SparkConf: map[string]string{
						"spark.databricks.delta.preview.enabled": "true",
					},
				}},
			},
			Tasks: []jobs.Task{
				{
					TaskKey: "notebook_task",
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: "/test/notebook",
					},
				},
			},
			Parameters: []jobs.JobParameterDefinition{
				{
					Name:    "empty",
					Default: "",
				},
			},
		},
	}, nil)

	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/test/notebook").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeNotebook,
		Language:   workspace.LanguagePython,
		Path:       "/test/notebook",
	}, nil)

	notebookContent := io.NopCloser(bytes.NewBufferString("# Databricks notebook source\nNotebook content"))
	workspaceApi.EXPECT().Download(mock.Anything, "/test/notebook", mock.Anything).Return(notebookContent, nil)

	cmd.SetContext(bundle.Context(context.Background(), b))
	require.NoError(t, cmd.Flag("existing-job-id").Value.Set("1234"))

	configDir := filepath.Join(root, "resources")
	require.NoError(t, cmd.Flag("config-dir").Value.Set(configDir))

	srcDir := filepath.Join(root, "src")
	require.NoError(t, cmd.Flag("source-dir").Value.Set(srcDir))

	var key string
	cmd.Flags().StringVar(&key, "key", "test_job", "")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "test_job.job.yml"))
	require.NoError(t, err)

	require.Equal(t, fmt.Sprintf(`resources:
  jobs:
    test_job:
      name: test-job
      job_clusters:
        - new_cluster:
            custom_tags:
              "Tag1": "24X7-1234"
        - new_cluster:
            spark_conf:
              "spark.databricks.delta.preview.enabled": "true"
      tasks:
        - task_key: notebook_task
          notebook_task:
            notebook_path: %s
      parameters:
        - name: empty
          default: ""
`, filepath.Join("..", "src", "notebook.py")), string(data))

	data, err = os.ReadFile(filepath.Join(srcDir, "notebook.py"))
	require.NoError(t, err)
	require.Equal(t, "# Databricks notebook source\nNotebook content", string(data))
}

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestGenerateJobCommandOldFileRename(t *testing.T) {
	cmd := NewGenerateJobCommand()

	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	jobsApi := m.GetMockJobsAPI()
	jobsApi.EXPECT().Get(mock.Anything, jobs.GetJobRequest{JobId: 1234}).Return(&jobs.Job{
		Settings: &jobs.JobSettings{
			Name: "test-job",
			JobClusters: []jobs.JobCluster{
				{NewCluster: compute.ClusterSpec{
					CustomTags: map[string]string{
						"Tag1": "24X7-1234",
					},
				}},
				{NewCluster: compute.ClusterSpec{
					SparkConf: map[string]string{
						"spark.databricks.delta.preview.enabled": "true",
					},
				}},
			},
			Tasks: []jobs.Task{
				{
					TaskKey: "notebook_task",
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: "/test/notebook",
					},
				},
			},
			Parameters: []jobs.JobParameterDefinition{
				{
					Name:    "empty",
					Default: "",
				},
			},
		},
	}, nil)

	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().GetStatusByPath(mock.Anything, "/test/notebook").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeNotebook,
		Language:   workspace.LanguagePython,
		Path:       "/test/notebook",
	}, nil)

	notebookContent := io.NopCloser(bytes.NewBufferString("# Databricks notebook source\nNotebook content"))
	workspaceApi.EXPECT().Download(mock.Anything, "/test/notebook", mock.Anything).Return(notebookContent, nil)

	cmd.SetContext(bundle.Context(context.Background(), b))
	require.NoError(t, cmd.Flag("existing-job-id").Value.Set("1234"))

	configDir := filepath.Join(root, "resources")
	require.NoError(t, cmd.Flag("config-dir").Value.Set(configDir))

	srcDir := filepath.Join(root, "src")
	require.NoError(t, cmd.Flag("source-dir").Value.Set(srcDir))

	var key string
	cmd.Flags().StringVar(&key, "key", "test_job", "")

	// Create an old generated file first
	oldFilename := filepath.Join(configDir, "test_job.yml")
	touchEmptyFile(t, oldFilename)

	// Having an existing files require --force flag to regenerate them
	require.NoError(t, cmd.Flag("force").Value.Set("true"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Make sure file do not exists after the run
	_, err = os.Stat(oldFilename)
	require.ErrorIs(t, err, fs.ErrNotExist)

	data, err := os.ReadFile(filepath.Join(configDir, "test_job.job.yml"))
	require.NoError(t, err)

	require.Equal(t, fmt.Sprintf(`resources:
  jobs:
    test_job:
      name: test-job
      job_clusters:
        - new_cluster:
            custom_tags:
              "Tag1": "24X7-1234"
        - new_cluster:
            spark_conf:
              "spark.databricks.delta.preview.enabled": "true"
      tasks:
        - task_key: notebook_task
          notebook_task:
            notebook_path: %s
      parameters:
        - name: empty
          default: ""
`, filepath.Join("..", "src", "notebook.py")), string(data))

	data, err = os.ReadFile(filepath.Join(srcDir, "notebook.py"))
	require.NoError(t, err)
	require.Equal(t, "# Databricks notebook source\nNotebook content", string(data))
}
