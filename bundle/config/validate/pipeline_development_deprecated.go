package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const (
	pipelineDevelopmentDeprecatedSummary = `The "development" property of pipelines is deprecated. Use "mode: development" in your databricks.yml instead. See https://docs.databricks.com/dev-tools/bundles/deployment-modes.`

	pipelineDevelopmentIgnoredSummary = `The "development: false" set by a Python mutator is ignored when running the pipeline. "bundle run" uses development mode from "mode: development" instead.`
	pipelineDevelopmentIgnoredDetail  = `To run this pipeline in production, set "presets.pipelines_development: false" on the target.`
)

type pipelineDevelopmentDeprecated struct{ bundle.RO }

// PipelineDevelopmentDeprecated warns when the development property is set on a pipeline.
func PipelineDevelopmentDeprecated() bundle.ReadOnlyMutator {
	return &pipelineDevelopmentDeprecated{}
}

func (v *pipelineDevelopmentDeprecated) Name() string {
	return "validate:pipeline_development_deprecated"
}

func (v *pipelineDevelopmentDeprecated) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	presetEnabled := config.IsExplicitlyEnabled(b.Config.Presets.PipelinesDevelopment)

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("development"))
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Only user-written values have a location; the value set by "mode: development" does not.
		if len(v.Locations()) == 0 {
			return v, nil
		}

		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   pipelineDevelopmentDeprecatedSummary,
			Locations: v.Locations(),
			Paths:     []dyn.Path{p},
		})

		// The preset overwrites YAML values with true, so a false here was set by a Python
		// mutator after the preset ran. "bundle run" still sends development: true.
		if development, ok := v.AsBool(); ok && presetEnabled && !development {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   pipelineDevelopmentIgnoredSummary,
				Detail:    pipelineDevelopmentIgnoredDetail,
				Locations: v.Locations(),
				Paths:     []dyn.Path{p},
			})
		}
		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
