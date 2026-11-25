package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type referenceableResourceKeys struct{}

// ReferenceableResourceKeys rejects a resource with permissions or grants whose key
// cannot appear in a ${...} reference. The direct engine links permissions and grants
// to their resource with such a reference, so planning them would fail.
func ReferenceableResourceKeys() bundle.Mutator {
	return &referenceableResourceKeys{}
}

func (*referenceableResourceKeys) Name() string {
	return "validate:referenceable_resource_keys"
}

func (*referenceableResourceKeys) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if dynvar.IsPureVariableReference("${" + p.String() + "}") {
			return v, nil
		}
		for _, field := range []string{"permissions", "grants"} {
			// An empty list plans nothing, so it needs no reference.
			if items, ok := v.Get(field).AsSequence(); !ok || len(items) == 0 {
				continue
			}
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("resource key %q cannot be referenced, so its %s cannot be deployed", p[len(p)-1].Key(), field),
				Detail: "Permissions and grants are linked to their resource with a ${resources...} reference. " +
					"Rename the key to use only letters, digits, hyphens and underscores, with at least one letter, " +
					"ending with a letter or a digit.",
				Locations: v.Locations(),
				Paths:     dyn.ToStructPaths(p),
			})
		}
		return v, nil
	})
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	// Resources are visited in random order.
	slices.SortStableFunc(diags, func(a, b diag.Diagnostic) int {
		return cmp.Compare(a.Paths[0].String(), b.Paths[0].String())
	})
	return diags
}
