package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

type tfOnlyReferences struct{}

// TFOnlyReferences validates that no cross-resource references point to
// Terraform-only fields (fields that exist in TF schema but have no DABs equivalent).
// The direct engine cannot resolve such references at deploy time.
func TFOnlyReferences() bundle.Mutator {
	return &tfOnlyReferences{}
}

func (m *tfOnlyReferences) Name() string {
	return "validate:tf_only_references"
}

func (m *tfOnlyReferences) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
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
			if d := checkTFOnlyReference(r, v.Location()); d != nil {
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
func checkTFOnlyReference(ref string, loc diag.Location) *diag.Diagnostic {
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

	return &diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%q: Terraform-only field; cross-resource references to Terraform-only fields are not supported by the direct engine", ref),
		Locations: []diag.Location{loc},
	}
}
