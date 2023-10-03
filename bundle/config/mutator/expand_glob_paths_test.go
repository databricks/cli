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
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
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
							},
						},
					},
				},
			},
		},
	}

	m := ExpandGlobPaths()
	err := bundle.Apply(context.Background(), b, m)
	require.NoError(t, err)

	libraries := b.Config.Resources.Pipelines["pipeline"].Libraries
	require.Len(t, libraries, 5)
	require.True(t, containsNotebook(libraries, "test/test2.ipynb"))
	require.True(t, containsNotebook(libraries, "test/test3.ipynb"))
	require.True(t, containsJar(libraries, "test1.jar"))
	require.True(t, containsFile(libraries, "test/test2.py"))
	require.True(t, containsFile(libraries, "test/test3.py"))
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

func containsFile(libraries []pipelines.PipelineLibrary, path string) bool {
	for _, l := range libraries {
		if l.File != nil && l.File.Path == path {
			return true
		}
	}

	return false
}
