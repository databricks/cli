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

// ValidateEngine validates that the bundle.deployment.engine setting is valid.
func ValidateEngine() bundle.ReadOnlyMutator {
	return &validateEngine{}
}

func (v *validateEngine) Name() string {
	return "validate:engine"
}

func (v *validateEngine) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	configEngine := b.Config.Bundle.Deployment.Engine
	if configEngine == engine.EngineNotSet {
		return nil
	}

	if _, ok := engine.Parse(string(configEngine)); !ok {
		val := dyn.GetValue(b.Config.Value(), "bundle.deployment.engine")
		loc := val.Location()
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("invalid value %q for bundle.deployment.engine (expected %q or %q)", configEngine, engine.EngineTerraform, engine.EngineDirect),
			Locations: []dyn.Location{loc},
		}}
	}

	return nil
}
