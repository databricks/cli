package resourcemutator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
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
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("./**/*.ipynb"),
									},
								},
								{
									Jar: new("./*.jar"),
								},
								{
									File: &pipelines.NotebookLibrary{
										Path: new("./test/*.py"),
									},
								},
								{
									// This value is annotated to be defined in the "./relative" directory.
									File: &pipelines.NotebookLibrary{
										Path: new("./*.py"),
									},
								},
								{
									Maven: &pipelines.PipelinesMavenLibrary{
										Coordinates: new("org.jsoup:jsoup:1.7.2"),
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("./test1.ipynb"),
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("/Workspace/Users/me@company.com/test.ipynb"),
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("dbfs:/me@company.com/test.ipynb"),
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("/Repos/somerepo/test.ipynb"),
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: new("./non-existent.ipynb"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []diag.Location{{File: filepath.Join(dir, "resource.yml")}})
	bundletest.SetLocation(b, "resources.pipelines.pipeline.libraries[3]", []diag.Location{{File: filepath.Join(dir, "relative", "resource.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), resourcemutator.ExpandPipelineGlobPaths())
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
			paths = append(paths, utils.Deref(l.Notebook.Path))
		}
	}

	return paths
}

func collectJars(libraries []pipelines.PipelineLibrary) []string {
	var paths []string
	for _, l := range libraries {
		if utils.Deref(l.Jar) != "" {
			paths = append(paths, utils.Deref(l.Jar))
		}
	}

	return paths
}

func collectMaven(libraries []pipelines.PipelineLibrary) []string {
	var coordinates []string
	for _, l := range libraries {
		if l.Maven != nil {
			coordinates = append(coordinates, utils.Deref(l.Maven.Coordinates))
		}
	}

	return coordinates
}

func collectFiles(libraries []pipelines.PipelineLibrary) []string {
	var paths []string
	for _, l := range libraries {
		if l.File != nil {
			paths = append(paths, utils.Deref(l.File.Path))
		}
	}

	return paths
}
