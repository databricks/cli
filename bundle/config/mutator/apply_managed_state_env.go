package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	envlib "github.com/databricks/cli/libs/env"
)

type applyManagedStateEnv struct{}

// ApplyManagedStateEnv reads DATABRICKS_BUNDLE_MANAGED_STATE and writes it
// into bundle.deployment.managed_state when the field is not already set in
// configuration. Configuration takes priority over the environment variable.
// Only the value "true" enables managed state; any other value is ignored.
func ApplyManagedStateEnv() bundle.Mutator {
	return &applyManagedStateEnv{}
}

func (m *applyManagedStateEnv) Name() string {
	return "ApplyManagedStateEnv"
}

func (m *applyManagedStateEnv) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Bundle.Deployment.ManagedState != nil {
		return nil
	}
	if envlib.Get(ctx, bundleenv.ManagedStateVariable) != "true" {
		return nil
	}
	enabled := true
	b.Config.Bundle.Deployment.ManagedState = &enabled
	return nil
}
