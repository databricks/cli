package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type configureClusterPolicyDefinition struct{}

func ConfigureClusterPolicyDefinition() bundle.Mutator {
	return &configureClusterPolicyDefinition{}
}

func (c configureClusterPolicyDefinition) Name() string {
	return "ConfigureClusterPolicyDefinition"
}

func (c configureClusterPolicyDefinition) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	policies := b.Config.Resources.ClusterPolicies
	for _, name := range slices.Sorted(maps.Keys(policies)) {
		policy := policies[name]
		if policy == nil {
			continue
		}
		fields := []struct {
			key   string
			value any
		}{
			{"definition", policy.Definition},
			{"policy_family_definition_overrides", policy.PolicyFamilyDefinitionOverrides},
		}
		for _, f := range fields {
			fieldPath := structpath.NewPath(nil, "resources", "cluster_policies", name, f.key)

			// Marshal an inline structured value to a JSON string so both
			// config-side and state-side carry the same plain string. Otherwise
			// YAML decodes small ints as Go `int` while state JSON round-trip
			// decodes them as `float64`, and structdiff reports false drift.
			// An absent value is left for backend validation.
			s, ok, kind, err := inlineJSON(f.value)
			if err != nil {
				return diags.Extend(diag.FromErr(fmt.Errorf("failed to marshal inline %s: %w", f.key, err)))
			}
			if kind != "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("%s must be a string or map, got %s", f.key, kind),
					Locations: b.Config.GetLocationsOfExact(fieldPath),
				})
				continue
			}
			if ok {
				if err := b.Config.Set(fieldPath, s); err != nil {
					return diags.Extend(diag.FromErr(err))
				}
			}
		}
	}

	return diags
}
