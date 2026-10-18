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
		d, err := inlinePolicyField(b, name, "definition", policy.Definition)
		diags = diags.Extend(d)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
		d, err = inlinePolicyField(b, name, "policy_family_definition_overrides", policy.PolicyFamilyDefinitionOverrides)
		diags = diags.Extend(d)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
	}

	return diags
}

// inlinePolicyField marshals the inline structured value of the field key of the
// cluster policy name to a JSON string, so both config-side and state-side carry the
// same plain string. Otherwise YAML decodes small ints as Go `int` while state JSON
// round-trip decodes them as `float64`, and structdiff reports false drift. An absent
// value is left for backend validation.
func inlinePolicyField(b *bundle.Bundle, name, key string, value any) (diag.Diagnostics, error) {
	fieldPath := structpath.NewPath(nil, "resources", "cluster_policies", name, key)
	s, ok, kind, err := inlineJSON(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inline %s: %w", key, err)
	}
	if kind != "" {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("%s must be a string or map, got %s", key, kind),
			Locations: b.Config.GetLocationsOfExact(fieldPath),
		}}, nil
	}
	if !ok {
		return nil, nil
	}
	return nil, b.Config.Set(fieldPath, s)
}
