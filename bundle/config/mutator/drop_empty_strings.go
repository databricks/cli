package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/validation/generated"
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
		keep, err := requiredEmptyStrings(root)
		if err != nil {
			return root, err
		}
		root, err = dyn.Map(root, "resources", func(_ dyn.Path, resources dyn.Value) (dyn.Value, error) {
			// Normalize against the resources type so omitempty is known per field.
			// Only DropEmptyStrings is set: existing values are kept as-is otherwise.
			out, normDiags := convert.Normalize(config.Resources{}, resources, convert.DropEmptyStrings)
			diags = diags.Extend(normDiags)
			return out, nil
		})
		if err != nil {
			return root, err
		}
		for _, kv := range keep {
			root, err = dyn.SetByPath(root, kv.path, kv.value)
			if err != nil {
				return root, err
			}
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

// requiredEmptyStrings returns the paths of empty strings set on required
// fields. Those are kept: "" is a value the user chose for a field that must be
// sent, as opposed to an optional field left empty. The sdk-go based resource
// types mark every field omitempty, so required fields come from the generated
// validation metadata instead of struct tags.
func requiredEmptyStrings(root dyn.Value) ([]pathValue, error) {
	trie := &dyn.TrieNode{}
	for k := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return nil, err
		}
		if err := trie.Insert(pattern); err != nil {
			return nil, err
		}
	}
	var out []pathValue
	err := dyn.WalkReadOnly(root, func(p dyn.Path, v dyn.Value) error {
		pattern, ok := trie.SearchPath(p)
		if !ok {
			return nil
		}
		for _, field := range generated.RequiredFields[pattern.String()] {
			fv := v.Get(field)
			if s, ok := fv.AsString(); ok && s == "" {
				out = append(out, pathValue{p.Append(dyn.Key(field)), fv})
			}
		}
		return nil
	})
	return out, err
}

type pathValue struct {
	path  dyn.Path
	value dyn.Value
}
