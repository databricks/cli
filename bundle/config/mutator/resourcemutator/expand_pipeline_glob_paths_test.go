package resourcemutator_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
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

func TestExpandGlobPathsInPipelinesKeepsOrder(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "a/1.py"))
	touchEmptyFile(t, filepath.Join(dir, "a/2.py"))
	touchEmptyFile(t, filepath.Join(dir, "n.ipynb"))

	glob := &pipelines.FileLibrary{Path: "a/*.py"}
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Libraries: []pipelines.PipelineLibrary{
								{Notebook: &pipelines.NotebookLibrary{Path: "n.ipynb"}},
								{File: glob},
								{File: &pipelines.FileLibrary{Path: "/Workspace/x.py"}},
								{Notebook: &pipelines.NotebookLibrary{}},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.ExpandPipelineGlobPaths())
	require.NoError(t, diags.Error())

	assert.Equal(t, []pipelines.PipelineLibrary{
		{Notebook: &pipelines.NotebookLibrary{Path: "n.ipynb"}},
		{File: &pipelines.FileLibrary{Path: "a/1.py"}},
		{File: &pipelines.FileLibrary{Path: "a/2.py"}},
		{File: &pipelines.FileLibrary{Path: "/Workspace/x.py"}},
		// The empty notebook is kept.
		{Notebook: &pipelines.NotebookLibrary{}},
	}, b.Config.Resources.Pipelines["pipeline"].Libraries)

	// The original library is not modified.
	assert.Equal(t, "a/*.py", glob.Path)
}

func TestExpandGlobPathsInPipelinesWithoutLibraries(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"unset": {},
					"empty": {
						CreatePipeline: pipelines.CreatePipeline{
							Libraries: []pipelines.PipelineLibrary{},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.ExpandPipelineGlobPaths())
	require.NoError(t, diags.Error())

	assert.Nil(t, b.Config.Resources.Pipelines["unset"].Libraries)
	assert.Equal(t, []pipelines.PipelineLibrary{}, b.Config.Resources.Pipelines["empty"].Libraries)
}

func TestExpandGlobPathsInPipelinesKeepsLocations(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "plain.py"))
	touchEmptyFile(t, filepath.Join(dir, "nbs/a.py"))
	touchEmptyFile(t, filepath.Join(dir, "nbs/b.py"))
	touchEmptyFile(t, filepath.Join(dir, "nbs/c.py"))

	root, diags := config.LoadFromBytes("databricks.yml", []byte(`bundle:
  name: test-bundle
resources:
  pipelines:
    p:
      name: p
      libraries:
        - file:
            path: ./plain.py
        - notebook:
            path: ./nbs/*.py
        - file:
            path: ./plain.py
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{BundleRootPath: dir, Config: *root}

	diags = bundle.Apply(t.Context(), b, resourcemutator.ExpandPipelineGlobPaths())
	require.NoError(t, diags.Error())

	// Each expanded library has the location of the glob it comes from, and the
	// library after the glob keeps its own.
	var got []string
	for i, lib := range b.Config.Resources.Pipelines["p"].Libraries {
		field := "file"
		if lib.Notebook != nil {
			field = "notebook"
		}
		got = append(got, b.Config.GetLocation(fmt.Sprintf("resources.pipelines.p.libraries[%d].%s.path", i, field)).String())
	}
	assert.Equal(t, []string{
		"databricks.yml:9:19",
		"databricks.yml:11:19",
		"databricks.yml:11:19",
		"databricks.yml:11:19",
		"databricks.yml:13:19",
	}, got)
}
