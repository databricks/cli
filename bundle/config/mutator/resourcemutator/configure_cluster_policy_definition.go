package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	// Skip converting the configuration if there is nothing to configure.
	if len(b.Config.Resources.ClusterPolicies) == 0 {
		return nil
	}

	pattern := structpath.MustParsePattern("resources.cluster_policies.*")

	err := structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, v structvar.View) error {
		for _, field := range jsonPolicyFields {
			def := v.Get(field)

			// Marshal an inline structured value to a JSON string so both
			// config-side and state-side carry the same plain string. Otherwise
			// YAML decodes small ints as Go `int` while state JSON round-trip
			// decodes them as `float64`, and structdiff reports false drift.
			switch def.Kind() {
			case structvar.KindInvalid, structvar.KindNil, structvar.KindString:
				// KindInvalid means the field is absent; leave it for backend validation.
				continue
			case structvar.KindMap:
				jsonBytes, err := json.Marshal(def.AsAny())
				if err != nil {
					return fmt.Errorf("failed to marshal inline %s: %w", field, err)
				}
				err = b.Config.Set(structpath.NewStringKey(p, field), string(jsonBytes))
				if err != nil {
					return err
				}
			default:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("%s must be a string or map, got %s", field, def.Kind()),
					Locations: def.Locations(),
				})
			}
		}
		return nil
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
