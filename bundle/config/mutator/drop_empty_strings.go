package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

type dropEmptyStrings struct{}

// DropEmptyStrings removes empty-string values on omitempty resource fields, so
// they are not force-sent to the backend. An empty string reaches this point
// either literally (policy_id: "") or via a variable that resolved to "", so
// this must run after variable resolution.
//
// The direct engine converts the resolved config through convert.ToTyped, which
// force-sends explicitly-set zero values, defeating the omitempty tag. Dropping
// here fixes it and makes the result visible in `bundle validate -o json`, which
// serializes the dynamic value.
func DropEmptyStrings() bundle.Mutator {
	return &dropEmptyStrings{}
}

func (m *dropEmptyStrings) Name() string {
	return "DropEmptyStrings"
}

func (m *dropEmptyStrings) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		root, err := dyn.Map(root, "resources", func(_ dyn.Path, resources dyn.Value) (dyn.Value, error) {
			// Normalize against the resources type so omitempty is known per field.
			// Only DropEmptyStrings is set: existing values are kept as-is otherwise.
			out, normDiags := convert.Normalize(config.Resources{}, resources, convert.DropEmptyStrings)
			diags = diags.Extend(normDiags)
			return out, nil
		})
		if err != nil {
			return root, err
		}

		// Keep an empty apps description instead of dropping it: an app update without
		// the field leaves the remote description unchanged, so removing description
		// from config would never converge. Force-sending "" clears it.
		return dyn.MapByPattern(root, dyn.NewPattern(dyn.Key("resources"), dyn.Key("apps"), dyn.AnyKey()), func(_ dyn.Path, app dyn.Value) (dyn.Value, error) {
			if _, err := dyn.Get(app, "description"); err != nil {
				return dyn.Set(app, "description", dyn.V(""))
			}
			return app, nil
		})
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
