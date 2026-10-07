package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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
	view := b.Config.View()
	environments := view.Get("environments")
	targets := view.Get("targets")

	// Return an error if both "environments" and "targets" are set.
	if environments.IsValid() && targets.IsValid() {
		return diag.Errorf(
			"both 'environments' and 'targets' are specified; only 'targets' should be used: %s",
			environments.Location().String(),
		)
	}

	// Rewrite "environments" to "targets" and drop the "environments" key.
	if environments.IsValid() && !targets.IsValid() {
		err := b.Config.Assign(structpath.NewStringKey(nil, "targets"), environments)
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.FromErr(b.Config.Delete(structpath.NewStringKey(nil, "environments")))
	}

	return nil
}
