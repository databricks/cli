package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

type validateCatalogsOnlyWithDirect struct {
	engine engine.EngineType
}

// ValidateCatalogsOnlyWithDirect returns a mutator that validates catalog resources
// are only used with direct deployment mode.
func ValidateCatalogsOnlyWithDirect(engine engine.EngineType) bundle.Mutator {
	return &validateCatalogsOnlyWithDirect{engine: engine}
}

func (m *validateCatalogsOnlyWithDirect) Name() string {
	return "ValidateCatalogsOnlyWithDirect"
}

func (m *validateCatalogsOnlyWithDirect) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Catalogs are only supported in direct deployment mode
	if !m.engine.IsDirect() && len(b.Config.Resources.Catalogs) > 0 {
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Catalog resources are only supported with direct deployment mode",
			Detail: "Catalog resources require direct deployment mode. " +
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use catalog resources.\n" +
				"Learn more at https://docs.databricks.com/dev-tools/bundles/deployment-modes.html",
			Locations: b.Config.GetLocations("resources.catalogs"),
		}}
	}

	return nil
}
