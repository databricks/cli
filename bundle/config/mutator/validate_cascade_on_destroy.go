package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

type validateCascadeOnDestroy struct {
	engine engine.EngineType
}

// ValidateCascadeOnDestroy returns a mutator that errors when a pipeline sets
// cascade_on_destroy while using the terraform deployment engine. The terraform
// provider does not support the attribute yet, so it is only honored in direct
// deployment mode.
func ValidateCascadeOnDestroy(e engine.EngineType) bundle.Mutator {
	return &validateCascadeOnDestroy{engine: e}
}

func (m *validateCascadeOnDestroy) Name() string {
	return "ValidateCascadeOnDestroy"
}

func (m *validateCascadeOnDestroy) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for key, pipeline := range b.Config.Resources.Pipelines {
		if pipeline.CascadeOnDestroy == nil {
			continue
		}
		path := "resources.pipelines." + key + ".cascade_on_destroy"
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "cascade_on_destroy is only supported in direct deployment mode",
			Detail: "cascade_on_destroy is not yet supported by the terraform provider. " +
				"Set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' or set 'bundle.engine: direct' in your databricks.yml to use it.\n" +
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
			Locations: b.Config.GetLocations(path),
		})
	}
	return diags
}
