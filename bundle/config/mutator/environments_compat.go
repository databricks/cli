package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type environmentsToTargets struct{}

func EnvironmentsToTargets() bundle.Mutator {
	return &environmentsToTargets{}
}

func (m *environmentsToTargets) Name() string {
	return "EnvironmentsToTargets"
}

func (m *environmentsToTargets) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Short circuit if the "environments" key is not set.
	// This is the common case.
	if b.Config.Environments == nil {
		return nil
	}

	// The "environments" key is set; validate and rewrite it to "targets".
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		environments := v.Get("environments")
		targets := v.Get("targets")

		// Return an error if both "environments" and "targets" are set.
		if environments.Kind() != dyn.KindInvalid && targets.Kind() != dyn.KindInvalid {
			return dyn.InvalidValue, fmt.Errorf(
				"both 'environments' and 'targets' are specified; only 'targets' should be used: %s",
				environments.Location().String(),
			)
		}

		// Rewrite "environments" to "targets".
		if environments.Kind() != dyn.KindInvalid && targets.Kind() == dyn.KindInvalid {
			nv, err := dyn.Set(v, "targets", environments)
			if err != nil {
				return dyn.InvalidValue, err
			}
			// Drop the "environments" key.
			return dyn.Walk(nv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				switch len(p) {
				case 0:
					return v, nil
				case 1:
					if p[0] == dyn.Key("environments") {
						return v, dyn.ErrDrop
					}
				}
				return v, dyn.ErrSkip
			})
		}

		return v, nil
	})

	return diag.FromErr(err)
}
