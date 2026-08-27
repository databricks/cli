package validate_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredRejectsIncompletePipelineLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"weird[0]key": {
						CreatePipeline: pipelines.CreatePipeline{
							Name: "p",
							Libraries: []pipelines.PipelineLibrary{
								{File: &pipelines.FileLibrary{}},
								{Notebook: &pipelines.NotebookLibrary{}},
								{Glob: &pipelines.PathPattern{}},
								{File: &pipelines.FileLibrary{Path: "ok.py"}},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())
	assert.ElementsMatch(t, []string{
		"pipeline library file path is required",
		"pipeline library notebook path is required",
		"pipeline library glob include is required",
	}, diagSummaries(diags))
}

func TestRequiredPreservesPipelineLibraryLocationForMetacharacterResourceKey(t *testing.T) {
	location := dyn.Location{File: "databricks.yml", Line: 8, Column: 17}
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"pipelines": dyn.V(map[string]dyn.Value{
					"weird[0]key": dyn.V(map[string]dyn.Value{
						"libraries": dyn.V([]dyn.Value{
							dyn.V(map[string]dyn.Value{
								"file": dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{location}),
							}),
						}),
					}),
				}),
			}),
		}), nil
	}))

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.Len(t, diags, 1)
	assert.Equal(t, "pipeline library file path is required", diags[0].Summary)
	assert.Equal(t, []dyn.Location{location}, diags[0].Locations)
}

func TestRequiredAcceptsCompletePipelineLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Name: "pipeline",
							Libraries: []pipelines.PipelineLibrary{
								{File: &pipelines.FileLibrary{Path: "file.py"}},
								{Notebook: &pipelines.NotebookLibrary{Path: "notebook.py"}},
								{Glob: &pipelines.PathPattern{Include: "src/**"}},
							},
						},
					},
				},
			},
		},
	}

	assert.Empty(t, bundle.Apply(t.Context(), b, validate.Required()))
}
