package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type configureDashboardDefaults struct{}

func ConfigureDashboardDefaults() bundle.Mutator {
	return &configureDashboardDefaults{}
}

func (m *configureDashboardDefaults) Name() string {
	return "ConfigureDashboardDefaults"
}

func (m *configureDashboardDefaults) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
	)

	// Configure defaults for all dashboards.
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			var err error
			v, err = setIfNotExists(v, dyn.NewPath(dyn.Key("parent_path")), dyn.V(b.Config.Workspace.ResourcePath))
			if err != nil {
				return dyn.InvalidValue, err
			}
			v, err = setIfNotExists(v, dyn.NewPath(dyn.Key("embed_credentials")), dyn.V(false))
			if err != nil {
				return dyn.InvalidValue, err
			}
			return v, nil
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}

func setIfNotExists(v dyn.Value, path dyn.Path, defaultValue dyn.Value) (dyn.Value, error) {
	// Get the field at the specified path (if set).
	_, err := dyn.GetByPath(v, path)
	switch {
	case dyn.IsNoSuchKeyError(err):
		// OK, we'll set the default value.
		break
	case dyn.IsCannotTraverseNilError(err):
		// Cannot traverse the value, skip it.
		return v, nil
	case err == nil:
		// The field is set, skip it.
		return v, nil
	default:
		// Return the error.
		return v, err
	}

	// Set the field at the specified path.
	return dyn.SetByPath(v, path, defaultValue)
}
