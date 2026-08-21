package validate_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/validate"
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
