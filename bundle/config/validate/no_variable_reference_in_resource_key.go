package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	check := func(prefix *structpath.PathNode, r *config.Resources) {
		for _, group := range r.AllResources() {
			for _, key := range slices.Sorted(maps.Keys(group.Resources)) {
				if !structvar.ContainsVariableReference(key) {
					continue
				}
				p := structpath.NewPath(prefix, group.Description.PluralName, key)
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("resource key %q must not contain variable references", key),
					Paths:    []*structpath.PathNode{p},
				})
			}
		}
	}

	check(structpath.NewPath(nil, "resources"), &b.Config.Resources)
	for _, name := range slices.Sorted(maps.Keys(b.Config.Targets)) {
		if t := b.Config.Targets[name]; t != nil && t.Resources != nil {
			check(structpath.NewPath(nil, "targets", name, "resources"), t.Resources)
		}
	}

	return diags
}
