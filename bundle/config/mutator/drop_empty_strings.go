package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

		for key, d := range config.ResourceTypesImplementing[resources.EmptyStringSender]() {
			fields := d.FieldsSentWhenEmpty()
			root, err = dyn.MapByPattern(root, dyn.NewPattern(dyn.Key("resources"), dyn.Key(key), dyn.AnyKey()), func(_ dyn.Path, r dyn.Value) (dyn.Value, error) {
				for _, field := range fields {
					if _, err := dyn.Get(r, field); err != nil {
						r, err = dyn.Set(r, field, dyn.V(""))
						if err != nil {
							return r, err
						}
					}
				}
				return r, nil
			})
			if err != nil {
				return root, err
			}
		}
		return root, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
