package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type validateEngine struct{ bundle.RO }

// ValidateEngine validates that the bundle.engine setting is valid.
func ValidateEngine() bundle.ReadOnlyMutator {
	return &validateEngine{}
}

func (v *validateEngine) Name() string {
	return "validate:engine"
}

func (v *validateEngine) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	configEngine := b.Config.Bundle.Engine
	if configEngine == engine.EngineNotSet {
		return nil
	}

	loc := dyn.GetValue(b.Config.Value(), "bundle.engine").Location()

	parsed, ok := engine.Parse(string(configEngine))
	if !ok {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("invalid value %q for bundle.engine (expected %q)", configEngine, engine.EngineDirect),
			Locations: []dyn.Location{loc},
		}}
	}

	if parsed == engine.EngineTerraform {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "the Terraform deployment engine has been removed in Databricks CLI v1.19.0",
			Detail:    `Remove the "engine" setting (or set it to "direct") to deploy with the direct engine; existing Terraform state is migrated automatically. To keep using Terraform, downgrade to CLI v1.18.x. See https://docs.databricks.com/dev-tools/bundles/direct for details.`,
			Locations: []dyn.Location{loc},
		}}
	}

	return nil
}
