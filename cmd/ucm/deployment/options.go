package deployment

import (
	"context"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/phases"
)

// buildPhaseOptions is the indirection used by bind/unbind to assemble
// phases.Options. Tests overwrite this variable in-package to inject a fake
// TerraformFactory + DirectClientFactory and a local-disk Backend without
// standing up a real workspace client. Production callers get the default
// implementation which resolves the state backend from ctx + ucm config.
var buildPhaseOptions = defaultBuildPhaseOptions

// defaultBuildPhaseOptions is the production implementation of
// buildPhaseOptions. It mirrors cmd/ucm.defaultBuildPhaseOptions — duplicated
// here (not imported) to keep cmd/ucm/deployment free of a cyclic dependency
// on its parent package while still presenting the same test seam.
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
