package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const pipelineDevelopmentDeprecatedSummary = `The development property of pipelines is deprecated. Use the "mode: development" in your databricks.yml instead. See also https://docs.databricks.com/dev-tools/bundles/deployment-modes.`

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

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("development"))
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Only user-written values have a location; the value set by "mode: development" does not.
		if len(v.Locations()) > 0 {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   pipelineDevelopmentDeprecatedSummary,
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
