package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

type tfOnlyReferences struct{}

// TFOnlyReferences validates that no cross-resource references point to
// Terraform-only fields (fields that exist in TF schema but have no DABs equivalent).
//
// In direct mode this is an error because the direct engine cannot resolve such
// references at deploy time.  In Terraform mode it is a warning because the
// reference will break if the bundle is later migrated to the direct engine.
func TFOnlyReferences() bundle.Mutator {
	return &tfOnlyReferences{}
}

func (m *tfOnlyReferences) Name() string {
	return "validate:tf_only_references"
}

func (m *tfOnlyReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Resolve effective engine: config takes precedence over env var.
	effectiveEngine := b.Config.Bundle.Engine
	if effectiveEngine == engine.EngineNotSet {
		if envEngine, err := engine.FromEnv(ctx); err == nil {
			effectiveEngine = envEngine
		}
	}
	isDirect := effectiveEngine.IsDirect()

	var diags diag.Diagnostics

	// Walk the entire config looking for ${resources.*} references.
	_ = dyn.WalkReadOnly(b.Config.Value(), func(_ dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			if !strings.HasPrefix(r, "resources.") {
				continue
			}
			if d := checkTFOnlyReference(r, v.Location(), isDirect); d != nil {
				diags = append(diags, *d)
			}
		}
		return nil
	})

	if len(diags) > 0 {
		b.Metrics.AddBoolValue("has_tf_only_references", true)
	}

	return diags
}

// checkTFOnlyReference checks a single reference string like
// "resources.jobs.src.always_running" and returns a diagnostic when it refers
// to a TF-only field, or nil otherwise.
func checkTFOnlyReference(ref string, loc dyn.Location, isDirect bool) *diag.Diagnostic {
	p, err := dyn.NewPathFromString(ref)
	// Need at least resources.<group>.<name>.<field>
	if err != nil || len(p) < 4 || p[0].Key() != "resources" {
		return nil
	}

	group := p[1].Key()
	tfOnlyFields, ok := terraform_dabs_map.TerraformOnlyFields[group]
	if !ok || len(tfOnlyFields) == 0 {
		return nil
	}

	// Field path is everything after resources.<group>.<name>.
	fieldNode, err := structpath.ParsePath(p[3:].String())
	if err != nil {
		return nil
	}

	if !tfOnlyFields.Contains(fieldNode) {
		return nil
	}

	if isDirect {
		return &diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("%q: Terraform-only field; cross-resource references to Terraform-only fields are not supported by the direct engine", ref),
			Locations: []dyn.Location{loc},
		}
	}

	return &diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("%q: Terraform-only field; this reference will not work with the direct engine", ref),
		Locations: []dyn.Location{loc},
	}
}
