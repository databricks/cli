package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type noReferenceToSensitiveFields struct{}

// NoReferenceToSensitiveFields returns a validator that errors when any value
// in the config references a field tagged `bundle:"sensitive"` via interpolation
// (e.g. ${resources.secrets.my_secret.value}).
//
// Sensitive values must never appear in other fields: they would end up in
// validate JSON output, plan output, or debug logs where masking is impractical.
func NoReferenceToSensitiveFields() bundle.Mutator {
	return &noReferenceToSensitiveFields{}
}

func (m *noReferenceToSensitiveFields) Name() string {
	return "validate:no_reference_to_sensitive_fields"
}

func (m *noReferenceToSensitiveFields) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	_ = dyn.WalkReadOnly(b.Config.Value(), func(_ dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			if d := checkSensitiveReference(r, v.Locations()); d != nil {
				diags = append(diags, *d)
			}
		}
		return nil
	})

	return diags
}

// checkSensitiveReference returns an error diagnostic when the reference path
// points to a field tagged `bundle:"sensitive"`, or nil otherwise.
//
// It reads config.ResourcesTypes at call time (no extra cache) so tests can
// patch the map before calling and see the effect immediately.
func checkSensitiveReference(ref string, locs []dyn.Location) *diag.Diagnostic {
	if !strings.HasPrefix(ref, "resources.") {
		return nil
	}
	p, err := dyn.NewPathFromString(ref)
	// Need at least resources.<type>.<name>.<field>.
	if err != nil || len(p) < 4 {
		return nil
	}

	resourceType := p[1].Key()
	fieldName := p[3].Key()

	typ, ok := config.ResourcesTypes[resourceType]
	if !ok {
		return nil
	}
	sensitiveFields := convert.SensitiveFieldNames(typ)
	if !sensitiveFields[fieldName] {
		return nil
	}

	return &diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("${%s}: references a sensitive field and cannot be used in interpolation", ref),
		Locations: locs,
	}
}
