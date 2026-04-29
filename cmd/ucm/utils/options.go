package utils

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/phases"
)

// BuildPhaseOptionsHook is the package-level test seam used by ProcessUcm and
// every verb that needs phases.Options. Tests in cmd/ucm and
// cmd/ucm/deployment overwrite this variable to inject a fake
// TerraformFactory + DirectClientFactory and a local-disk Backend without
// standing up a real workspace client. Production callers get the default
// implementation which resolves the state backend from ctx + ucm config.
//
// Centralised here (rather than duplicated per cmd subpackage) so the seam
// matches TestProcessHook and there is a single place to swap the production
// wiring in tests.
var BuildPhaseOptionsHook = DefaultBuildPhaseOptions

// DefaultBuildPhaseOptions is the production implementation of
// BuildPhaseOptionsHook. It reads the workspace client off the Ucm struct
// (built lazily by ProcessUcm via MustConfigureUcm) and delegates the
// state-backend shape to ucm/deploy.BackendFromUcm. The returned Options
// always uses phases.DefaultTerraformFactory — the terraform wrapper
// constructor is expensive (binary resolution + working dir) and we only
// stand it up on first invocation.
func DefaultBuildPhaseOptions(ctx context.Context, u *ucm.Ucm) (phases.Options, error) {
	w, err := u.WorkspaceClientE()
	if err != nil {
		return phases.Options{}, err
	}
	backend, err := deploy.BackendFromUcm(ctx, u, w)
	if err != nil {
		return phases.Options{}, err
	}
	return phases.Options{
		Backend:          backend,
		TerraformFactory: phases.DefaultTerraformFactory,
	}, nil
}
