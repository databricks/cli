package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const definitionFieldName = "definition"

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
			def := v.Get(definitionFieldName)

			// Marshal an inline structured definition to a JSON string so both
			// config-side and state-side carry the same plain string. Otherwise
			// YAML decodes small ints as Go `int` while state JSON round-trip
			// decodes them as `float64`, and structdiff reports false drift.
			switch def.Kind() {
			case dyn.KindInvalid, dyn.KindNil, dyn.KindString:
				// KindInvalid means definition is absent; leave it for backend validation.
				return v, nil
			case dyn.KindMap, dyn.KindSequence:
				jsonBytes, err := json.Marshal(def.AsAny())
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to marshal inline definition: %w", err)
				}
				return dyn.Set(v, definitionFieldName, dyn.V(string(jsonBytes)))
			default:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("definition must be a string, map, or sequence, got %s", def.Kind()),
					Locations: def.Locations(),
				})
				return v, nil
			}
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
