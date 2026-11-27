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
	environmentsPath := structpath.NewStringKey(nil, "environments")
	targetsPath := structpath.NewStringKey(nil, "targets")

	// Return an error if both "environments" and "targets" are set.
	if b.Config.Targets != nil || b.Config.IsReferenceAt(targetsPath) {
		return diag.Errorf(
			"both 'environments' and 'targets' are specified; only 'targets' should be used: %s",
			b.Config.GetLocationOfExact(environmentsPath).String(),
		)
	}

	// Rewrite "environments" to "targets" and drop the "environments" key.
	b.Config.Targets = b.Config.Environments
	b.Config.Environments = nil
	b.Config.MoveMetadata(environmentsPath, targetsPath)
	return nil
}
