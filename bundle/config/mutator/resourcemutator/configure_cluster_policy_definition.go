package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// jsonPolicyFields are the JSON-policy fields normalized from inline YAML to a JSON string.
var jsonPolicyFields = []string{"definition", "policy_family_definition_overrides"}

type configureClusterPolicyDefinition struct{}

func ConfigureClusterPolicyDefinition() bundle.Mutator {
	return &configureClusterPolicyDefinition{}
}

func (c configureClusterPolicyDefinition) Name() string {
	return "ConfigureClusterPolicyDefinition"
}

func (c configureClusterPolicyDefinition) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("cluster_policies"),
		dyn.AnyKey(),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			for _, field := range jsonPolicyFields {
				def := v.Get(field)

				// Marshal an inline structured value to a JSON string so both
				// config-side and state-side carry the same plain string. Otherwise
				// YAML decodes small ints as Go `int` while state JSON round-trip
				// decodes them as `float64`, and structdiff reports false drift.
				switch def.Kind() {
				case dyn.KindInvalid, dyn.KindNil, dyn.KindString:
					// KindInvalid means the field is absent; leave it for backend validation.
					continue
				case dyn.KindMap:
					jsonBytes, err := json.Marshal(def.AsAny())
					if err != nil {
						return dyn.InvalidValue, fmt.Errorf("failed to marshal inline %s: %w", field, err)
					}
					v, err = dyn.Set(v, field, dyn.V(string(jsonBytes)))
					if err != nil {
						return dyn.InvalidValue, err
					}
				default:
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   fmt.Sprintf("%s must be a string or map, got %s", field, def.Kind()),
						Locations: def.Locations(),
					})
				}
			}
			return v, nil
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
