package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// ReservedResourceKey rejects a resource keyed "*". The selector "<type>.*" of
// --select means every resource of that type, so "*" cannot also name one resource.
func ReservedResourceKey() bundle.ReadOnlyMutator {
	return &reservedResourceKey{}
}

type reservedResourceKey struct{ bundle.RO }

func (v *reservedResourceKey) Name() string {
	return "validate:reserved_resource_key"
}

func (v *reservedResourceKey) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.Key("*")),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   `resource key "*" is reserved for --select '<type>.*'; rename this resource`,
				Locations: v.Locations(),
				Paths:     dyn.ToStructPaths(p),
			})
			return v, nil
		},
	)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
