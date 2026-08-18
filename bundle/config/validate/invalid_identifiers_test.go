package validate_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredRejectsEmptyAndControlCharIdentifiers(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Models: map[string]*resources.MlflowModel{
					"empty": {CreateModelRequest: ml.CreateModelRequest{Name: ""}},
				},
				Catalogs: map[string]*resources.Catalog{
					"blank": {CreateCatalog: catalog.CreateCatalog{Name: "   "}},
				},
				Volumes: map[string]*resources.Volume{
					"ctrl": {
						CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
							Name:        "tab\there",
							CatalogName: "main",
							SchemaName:  "default",
							VolumeType:  catalog.VolumeTypeManaged,
						},
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"nl": {
						CreateServingEndpoint: serving.CreateServingEndpoint{
							Name: "line1\nline2",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, validate.Required())
	require.True(t, diags.HasError())

	assert.ElementsMatch(t, []string{
		"catalog name must not be blank",
		"model name is required",
		"volume name must not contain control characters",
		"model_serving_endpoint name must not contain control characters",
	}, diagSummaries(diags))
}

func TestRequiredRejectsIncompletePipelineLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"p": {
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

func TestRequiredAcceptsValidIdentifiersAndPipelineLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Models: map[string]*resources.MlflowModel{
					"model": {CreateModelRequest: ml.CreateModelRequest{Name: "model"}},
				},
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

func TestRequiredAcceptsMissingOptionalUCParents(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				RegisteredModels: map[string]*resources.RegisteredModel{
					"model": {
						CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
							Name: "model",
						},
					},
				},
			},
		},
	}

	assert.Empty(t, bundle.Apply(t.Context(), b, validate.Required()))
}

func diagSummaries(diags diag.Diagnostics) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}
