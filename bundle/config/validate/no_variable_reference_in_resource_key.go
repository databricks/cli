package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type noVariableReferenceInResourceKey struct{}

// NoVariableReferenceInResourceKey validates that no resource key contains a variable reference.
// Resource keys are used as identifiers throughout the deployment pipeline and must be static strings.
func NoVariableReferenceInResourceKey() bundle.Mutator {
	return &noVariableReferenceInResourceKey{}
}

func (m *noVariableReferenceInResourceKey) Name() string {
	return "validate:no_variable_reference_in_resource_key"
}

func (m *noVariableReferenceInResourceKey) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey(), dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
	}

	for _, pattern := range patterns {
		_, err := dyn.MapByPattern(
			b.Config.Value(),
			pattern,
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				key := p[len(p)-1].Key()
				if dynvar.ContainsVariableReference(key) {
					diags = append(diags, diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   fmt.Sprintf("resource key %q must not contain variable references", key),
						Locations: v.Locations(),
						Paths:     []dyn.Path{p},
					})
				}
				return v, nil
			},
		)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	return diags
}
