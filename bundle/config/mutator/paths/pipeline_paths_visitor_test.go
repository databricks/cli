package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestVisitPipelinePaths(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"pipeline0": {
					CreatePipeline: &pipelines.CreatePipeline{
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
						},
					},
				},
			},
		},
	}

	actual := visitPipelinePaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.pipelines.pipeline0.libraries[0].file.path"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.libraries[1].notebook.path"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func visitPipelinePaths(t *testing.T, root config.Root) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitPipelinePaths(value, func(p dyn.Path, mode TranslateMode, v dyn.Value) (dyn.Value, error) {
			actual = append(actual, p)
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}
