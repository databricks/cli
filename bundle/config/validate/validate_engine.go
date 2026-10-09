package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type validateEngine struct{ bundle.RO }

// ValidateEngine validates that the bundle.engine setting is valid and warns
// about the bundle.terraform setting, which has no effect since the Terraform
// deployment engine was removed.
func ValidateEngine() bundle.ReadOnlyMutator {
	return &validateEngine{}
}

func (v *validateEngine) Name() string {
	return "validate:engine"
}

func (v *validateEngine) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	tf := b.Config.View().Lookup(structpath.MustParsePath("bundle.terraform"))
	if tf.IsValid() {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   "bundle.terraform is deprecated and has no effect: " + engine.TerraformRemovedSummary,
			Locations: tf.Locations(),
			Paths:     structpath.NewPathSlice("bundle", "terraform"),
		})
	}

	configEngine := b.Config.Bundle.Engine
	if configEngine == engine.EngineNotSet {
		return diags
	}

	parsed, ok := engine.Parse(string(configEngine))
	if !ok {
		return diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("invalid value %q for bundle.engine (expected %q)", configEngine, engine.EngineDirect),
			Paths:    structpath.NewPathSlice("bundle", "engine"),
		})
	}

	if parsed == engine.EngineTerraform {
		severity := diag.Error
		if b.AllowTerraformEngineConfig {
			severity = diag.Warning
		}
		return diags.Append(diag.Diagnostic{
			Severity: severity,
			Summary:  engine.TerraformRemovedSummary,
			Detail:   engine.TerraformRemovedConfigDetail,
			Paths:    structpath.NewPathSlice("bundle", "engine"),
			// Only the effective location; the automatic lookup would also add overridden ones.
			Locations: []diag.Location{b.Config.GetLocation("bundle.engine")},
		})
	}

	return diags
}
