package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type initializeVariables struct{}

// InitializeVariables initializes nil variables to their corresponding zero values.
// It runs while the configuration is loading, so that target overrides can set the
// fields of a variable that is declared without any.
func InitializeVariables() bundle.Mutator {
	return &initializeVariables{}
}

func (m *initializeVariables) Name() string {
	return "InitializeVariables"
}

func (m *initializeVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Get("variables").Kind() != dyn.KindMap {
			return v, nil
		}
		return dyn.Map(v, "variables", dyn.Foreach(func(_ dyn.Path, variable dyn.Value) (dyn.Value, error) {
			if variable.Kind() != dyn.KindNil {
				return variable, nil
			}
			return dyn.NewValue(map[string]dyn.Value{}, variable.Locations()), nil
		}))
	})
	return diag.FromErr(err)
}
