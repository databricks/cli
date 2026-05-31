package bundle

import (
	"context"

	bundleenv "github.com/databricks/cli/bundle/env"
	envlib "github.com/databricks/cli/libs/env"
)

// IsManagedState reports whether the bundle is opted into the deployment
// metadata service (DMS). Configuration takes priority over the
// DATABRICKS_BUNDLE_MANAGED_STATE environment variable.
func IsManagedState(ctx context.Context, b *Bundle) bool {
	if b.Config.Bundle.Deployment.ManagedState != nil {
		return *b.Config.Bundle.Deployment.ManagedState
	}
	return envlib.Get(ctx, bundleenv.ManagedStateVariable) == "true"
}
