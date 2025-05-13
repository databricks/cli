package resourcemutator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	assert "github.com/databricks/cli/libs/dyn/dynassert"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0o700)
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
	touchEmptyFile(t, filepath.Join(dir, "relative/test4.py"))
	touchEmptyFile(t, filepath.Join(dir, "relative/test5.py"))
	touchEmptyFile(t, filepath.Join(dir, "skip/test6.py"))
	touchEmptyFile(t, filepath.Join(dir, "skip/test7.py"))

	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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
										Path: "./test/*.py",
									},
								},
								{
									// This value is annotated to be defined in the "./relative" directory.
									File: &pipelines.FileLibrary{
										Path: "./*.py",
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
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./non-existent.ipynb",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})
	bundletest.SetLocation(b, "resources.pipelines.pipeline.libraries[3]", []dyn.Location{{File: filepath.Join(dir, "relative", "resource.yml")}})

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), resourcemutator.ExpandPipelineGlobPaths())
	require.NoError(t, diags.Error())

	libraries := b.Config.Resources.Pipelines["pipeline"].Libraries
	require.Len(t, libraries, 13)

	assert.ElementsMatch(
		t,
		collectNotebooks(libraries),
		[]string{
			// Making sure glob patterns are expanded correctly
			"test/test2.ipynb",
			"test/test3.ipynb",
			// Making sure exact file references work as well
			"test1.ipynb",
			// Making sure absolute pass to remote FS file references work as well
			"/Workspace/Users/me@company.com/test.ipynb",
			"dbfs:/me@company.com/test.ipynb",
			"/Repos/somerepo/test.ipynb",
			// Making sure other libraries are not replaced
			"non-existent.ipynb",
		},
	)

	assert.ElementsMatch(
		t,
		collectFiles(libraries),
		[]string{
			// Making sure glob patterns are expanded correctly
			"test/test2.py",
			"test/test3.py",
			// These patterns are defined relative to "./relative"
			"relative/test4.py",
			"relative/test5.py",
		},
	)

	// Making sure other libraries are not replaced
	assert.ElementsMatch(t, collectJars(libraries), []string{"./*.jar"})
	assert.ElementsMatch(t, collectMaven(libraries), []string{"org.jsoup:jsoup:1.7.2"})
}

func collectNotebooks(libraries []pipelines.PipelineLibrary) []string {
	var paths []string
	for _, l := range libraries {
		if l.Notebook != nil {
			paths = append(paths, l.Notebook.Path)
		}
	}

	return paths
}

func collectJars(libraries []pipelines.PipelineLibrary) []string {
	var paths []string
	for _, l := range libraries {
		if l.Jar != "" {
			paths = append(paths, l.Jar)
		}
	}

	return paths
}

func collectMaven(libraries []pipelines.PipelineLibrary) []string {
	var coordinates []string
	for _, l := range libraries {
		if l.Maven != nil {
			coordinates = append(coordinates, l.Maven.Coordinates)
		}
	}

	return coordinates
}

func collectFiles(libraries []pipelines.PipelineLibrary) []string {
	var paths []string
	for _, l := range libraries {
		if l.File != nil {
			paths = append(paths, l.File.Path)
		}
	}

	return paths
}
