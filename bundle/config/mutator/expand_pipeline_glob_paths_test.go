package mutator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestExpandGlobPathsInPipelines(t *testing.T) {
	dir := t.TempDir()

	touchEmptyFile(t, filepath.Join(dir, "test1.ipynb"))
	touchEmptyFile(t, filepath.Join(dir, "test/test2.ipynb"))
	touchEmptyFile(t, filepath.Join(dir, "test/test3.ipynb"))
	touchEmptyFile(t, filepath.Join(dir, "test1.jar"))
	touchEmptyFile(t, filepath.Join(dir, "test/test2.jar"))
	touchEmptyFile(t, filepath.Join(dir, "test/test3.jar"))
	touchEmptyFile(t, filepath.Join(dir, "test1.py"))
	touchEmptyFile(t, filepath.Join(dir, "test/test2.py"))
	touchEmptyFile(t, filepath.Join(dir, "test/test3.py"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: paths.Paths{
							LocalConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./**/*.ipynb",
									},
								},
								{
									Jar: "./*.jar",
								},
								{
									File: &pipelines.FileLibrary{
										Path: "./**/*.py",
									},
								},
								{
									Maven: &compute.MavenLibrary{
										Coordinates: "org.jsoup:jsoup:1.7.2",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./test1.ipynb",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "/Workspace/Users/me@company.com/test.ipynb",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "dbfs:/me@company.com/test.ipynb",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "/Repos/somerepo/test.ipynb",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	m := ExpandPipelineGlobPaths()
	err := bundle.Apply(context.Background(), b, m)
	require.NoError(t, err)

	libraries := b.Config.Resources.Pipelines["pipeline"].Libraries
	require.Len(t, libraries, 10)

	// Making sure glob patterns are expanded correctly
	require.True(t, containsNotebook(libraries, filepath.Join("test", "test2.ipynb")))
	require.True(t, containsNotebook(libraries, filepath.Join("test", "test3.ipynb")))
	require.True(t, containsFile(libraries, filepath.Join("test", "test2.py")))
	require.True(t, containsFile(libraries, filepath.Join("test", "test3.py")))

	// Making sure exact file references work as well
	require.True(t, containsNotebook(libraries, "test1.ipynb"))

	// Making sure absolute pass to remote FS file references work as well
	require.True(t, containsNotebook(libraries, "/Workspace/Users/me@company.com/test.ipynb"))
	require.True(t, containsNotebook(libraries, "dbfs:/me@company.com/test.ipynb"))
	require.True(t, containsNotebook(libraries, "/Repos/somerepo/test.ipynb"))

	// Making sure other libraries are not replaced
	require.True(t, containsJar(libraries, "./*.jar"))
	require.True(t, containsMaven(libraries, "org.jsoup:jsoup:1.7.2"))
}

func containsNotebook(libraries []pipelines.PipelineLibrary, path string) bool {
	for _, l := range libraries {
		if l.Notebook != nil && l.Notebook.Path == path {
			return true
		}
	}

	return false
}

func containsJar(libraries []pipelines.PipelineLibrary, path string) bool {
	for _, l := range libraries {
		if l.Jar == path {
			return true
		}
	}

	return false
}

func containsMaven(libraries []pipelines.PipelineLibrary, coordinates string) bool {
	for _, l := range libraries {
		if l.Maven != nil && l.Maven.Coordinates == coordinates {
			return true
		}
	}

	return false
}

func containsFile(libraries []pipelines.PipelineLibrary, path string) bool {
	for _, l := range libraries {
		if l.File != nil && l.File.Path == path {
			return true
		}
	}

	return false
}
