package validate

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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

	pipelines := b.Config.Resources.Pipelines
	for _, key := range slices.Sorted(maps.Keys(pipelines)) {
		pipeline := pipelines[key]
		//nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
		if pipeline == nil || (!pipeline.Development && !slices.Contains(pipeline.ForceSendFields, "Development")) {
			continue
		}

		p := structpath.NewStringKeys(nil, "resources", "pipelines", key, "development")

		// Only user-written values have a location; the value set by "mode: development" does not.
		locs := b.Config.LocationsAt(p)
		if len(locs) == 0 {
			continue
		}

		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   pipelineDevelopmentDeprecatedSummary,
			Locations: locs,
			Paths:     []*structpath.PathNode{p},
		})

		// The preset overwrites YAML values with true, so a false here was set by a Python
		// mutator after the preset ran. "bundle run" still sends development: true.
		if presetEnabled && !pipeline.Development { //nolint:staticcheck // SA1019: see above
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   pipelineDevelopmentIgnoredSummary,
				Detail:    pipelineDevelopmentIgnoredDetail,
				Locations: locs,
				Paths:     []*structpath.PathNode{p},
			})
		}
	}

	return diags
}
