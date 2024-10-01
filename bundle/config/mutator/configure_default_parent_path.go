package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type configureDefaultParentPath struct{}

func ConfigureDefaultParentPath() bundle.Mutator {
	return &configureDefaultParentPath{}
}

func (m *configureDefaultParentPath) Name() string {
	return "ConfigureDefaultParentPath"
}

func (m *configureDefaultParentPath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
	)

	// Default value for the parent path.
	defaultValue := b.Config.Workspace.ResourcePath

	// Configure the default parent path for all dashboards.
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Get the current parent path (if set).
			f, err := dyn.Get(v, "parent_path")
			switch {
			case dyn.IsNoSuchKeyError(err):
				// OK, we'll set the default value.
				break
			case dyn.IsCannotTraverseNilError(err):
				// Cannot traverse the value, skip it.
				return v, nil
			default:
				// Return the error.
				return v, err
			}

			// Configure the default value (if not set).
			if !f.IsValid() {
				f = dyn.V(defaultValue)
			}

			// Set the parent path.
			return dyn.Set(v, "parent_path", f)
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
