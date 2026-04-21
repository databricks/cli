package ucm

import (
	"context"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/phases"
)

// buildPhaseOptions is the indirection used by plan/deploy/destroy to assemble
// phases.Options. Tests overwrite this variable in-package to inject a fake
// TerraformFactory and a local-disk Backend without standing up a real
// workspace client. Production callers get the default implementation which
// resolves the state backend from ctx + ucm config.
var buildPhaseOptions = defaultBuildPhaseOptions

// defaultBuildPhaseOptions is the production implementation of
// buildPhaseOptions. It reads the workspace client from ctx (populated by
// root.MustWorkspaceClient when the user supplies auth) and delegates the
// state-backend shape to ucm/deploy.BackendFromUcm. The returned Options
// always uses phases.DefaultTerraformFactory — the terraform wrapper
// constructor is expensive (binary resolution + working dir) and we only
// stand it up on first invocation.
func defaultBuildPhaseOptions(ctx context.Context, u *ucm.Ucm) (phases.Options, error) {
	w := cmdctx.WorkspaceClient(ctx)
	backend, err := deploy.BackendFromUcm(ctx, u, w)
	if err != nil {
		return phases.Options{}, err
	}
	return phases.Options{
		Backend:          backend,
		TerraformFactory: phases.DefaultTerraformFactory,
	}, nil
}
