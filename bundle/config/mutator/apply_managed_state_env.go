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
	v, ok := envlib.GetBool(ctx, bundleenv.ManagedStateVariable)
	if !ok {
		return nil
	}
	b.Config.Bundle.Deployment.ManagedState = &v
	return nil
}
