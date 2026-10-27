package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestVisitPipelinePaths(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"pipeline0": {
					CreatePipeline: pipelines.CreatePipeline{
						RootPath: "src",
						Libraries: []pipelines.PipelineLibrary{
							{
								File: &pipelines.FileLibrary{
									Path: "dist/foo.whl",
								},
							},
							{
								Notebook: &pipelines.NotebookLibrary{
									Path: "src/foo.py",
								},
							},
							{
								Glob: &pipelines.PathPattern{
									Include: "a/b/c/**",
								},
							},
						},
						Environment: &pipelines.PipelinesEnvironment{
							Dependencies: []string{
								"src/foo.whl",
							},
						},
					},
					Testing: &resources.PipelineTesting{
						Sources: []resources.PipelineTestSource{
							{Glob: &pipelines.PathPattern{Include: "tests/**"}},
						},
						Environment: &resources.PipelineTestEnvironment{
							Dependencies: []string{
								"-r ./tests/requirements.txt",
								"dist/test_helpers.whl",
								"faker",
							},
						},
					},
				},
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitPipelinePaths)
	actual = append(actual, collectVisitedPaths(t, root, VisitPipelineLibrariesPaths)...)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.pipelines.pipeline0.libraries[0].file.path"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.libraries[1].notebook.path"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.libraries[2].glob.include"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.root_path"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.environment.dependencies[0]"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.testing.sources[0].glob.include"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.testing.environment.dependencies[0]"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.testing.environment.dependencies[1]"),
	}

	assert.ElementsMatch(t, expected, actual)
}
