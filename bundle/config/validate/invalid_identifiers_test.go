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

	diags := validate.Required().Apply(t.Context(), b)
	require.True(t, diags.HasError())

	summaries := diagSummaries(diags)
	assert.Contains(t, summaries, "model name is required")
	assert.Contains(t, summaries, "volume name must not contain control characters")
	assert.Contains(t, summaries, "model_serving_endpoint name must not contain control characters")
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

	diags := validate.Required().Apply(t.Context(), b)
	require.True(t, diags.HasError())

	summaries := diagSummaries(diags)
	assert.Contains(t, summaries, "pipeline library file path is required")
	assert.Contains(t, summaries, "pipeline library notebook path is required")
	assert.Contains(t, summaries, "pipeline library glob include is required")
}

func diagSummaries(diags diag.Diagnostics) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}
