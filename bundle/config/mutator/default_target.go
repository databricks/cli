package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type defineDefaultTarget struct {
	name string
}

// DefineDefaultTarget adds a target named "default"
// to the configuration if none have been defined. It runs while the configuration
// is loading.
func DefineDefaultTarget() bundle.Mutator {
	return &defineDefaultTarget{
		name: "default",
	}
}

func (m *defineDefaultTarget) Name() string {
	return fmt.Sprintf("DefineDefaultTarget(%s)", m.name)
}

func (m *defineDefaultTarget) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		targets := v.Get("targets")
		if m, ok := targets.AsMap(); ok && m.Len() > 0 {
			return v, nil
		}
		return dyn.Set(v, "targets", dyn.NewValue(map[string]dyn.Value{
			m.name: dyn.V(map[string]dyn.Value{}),
		}, targets.Locations()))
	})
	return diag.FromErr(err)
}
