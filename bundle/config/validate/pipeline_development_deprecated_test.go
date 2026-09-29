package validate_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

func loadPipelineDevelopmentBundle(t *testing.T, yaml string) *bundle.Bundle {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
	require.NoError(t, diags.Error())
	return &bundle.Bundle{Config: *root}
}

func TestPipelineDevelopmentDeprecatedUnset(t *testing.T) {
	b := loadPipelineDevelopmentBundle(t, `
resources:
  pipelines:
    my_pipeline:
      name: my_pipeline
`)
	diags := validate.PipelineDevelopmentDeprecated().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestPipelineDevelopmentDeprecatedSet(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			b := loadPipelineDevelopmentBundle(t, `
resources:
  pipelines:
    my_pipeline:
      name: my_pipeline
      development: `+value+`
`)
			diags := validate.PipelineDevelopmentDeprecated().Apply(t.Context(), b)
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Warning, diags[0].Severity)
			assert.Contains(t, diags[0].Summary, "The development property of pipelines is deprecated")
			assert.Equal(t, []dyn.Location{{File: "databricks.yml", Line: 6, Column: 20}}, diags[0].Locations)
			assert.Equal(t, []dyn.Path{dyn.MustPathFromString("resources.pipelines.my_pipeline.development")}, diags[0].Paths)
		})
	}
}

func TestPipelineDevelopmentDeprecatedTargetOverride(t *testing.T) {
	b := loadPipelineDevelopmentBundle(t, `
resources:
  pipelines:
    my_pipeline:
      name: my_pipeline

targets:
  prod:
    resources:
      pipelines:
        my_pipeline:
          development: true
`)
	require.NoError(t, b.Config.MergeTargetOverrides("prod"))

	diags := validate.PipelineDevelopmentDeprecated().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, []dyn.Location{{File: "databricks.yml", Line: 12, Column: 24}}, diags[0].Locations)
}

func applyDevelopmentMode(t *testing.T, b *bundle.Bundle) {
	bundle.ApplyFuncContext(t.Context(), b, func(_ context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &config.User{
			ShortName: "lennart",
			User:      &iam.User{UserName: "lennart@company.com", Id: "1"},
		}
	})
	b.Tagging = tags.ForCloud(&sdkconfig.Config{Host: "https://company.cloud.databricks.com"})

	diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ApplyTargetMode(), resourcemutator.ApplyPresets())
	require.NoError(t, diags.Error())
	require.True(t, b.Config.Resources.Pipelines["my_pipeline"].Development)
}

func TestPipelineDevelopmentDeprecatedNotSetByDevelopmentMode(t *testing.T) {
	b := loadPipelineDevelopmentBundle(t, `
bundle:
  mode: development

resources:
  pipelines:
    my_pipeline:
      name: my_pipeline
`)
	applyDevelopmentMode(t, b)

	diags := validate.PipelineDevelopmentDeprecated().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestPipelineDevelopmentDeprecatedOverwrittenByDevelopmentMode(t *testing.T) {
	b := loadPipelineDevelopmentBundle(t, `
bundle:
  mode: development

resources:
  pipelines:
    my_pipeline:
      name: my_pipeline
      development: false
`)
	applyDevelopmentMode(t, b)

	diags := validate.PipelineDevelopmentDeprecated().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, []dyn.Location{{File: "databricks.yml", Line: 9, Column: 20}}, diags[0].Locations)
}
