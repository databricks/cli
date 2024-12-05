package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type configureVolumeDefaults struct{}

func ConfigureVolumeDefaults() bundle.Mutator {
	return &configureVolumeDefaults{}
}

func (m *configureVolumeDefaults) Name() string {
	return "ConfigureVolumeDefaults"
}

func (m *configureVolumeDefaults) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("volumes"),
		dyn.AnyKey(),
	)

	// Configure defaults for all volumes.
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			var err error
			v, err = setIfNotExists(v, dyn.NewPath(dyn.Key("volume_type")), dyn.V("MANAGED"))
			if err != nil {
				return dyn.InvalidValue, err
			}
			return v, nil
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
