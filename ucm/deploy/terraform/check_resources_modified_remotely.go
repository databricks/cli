package terraform

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type checkResourcesModifiedRemotely struct {
	kinds []string
}

func (l *checkResourcesModifiedRemotely) Name() string {
	return "CheckResourcesModifiedRemotely"
}

func (l *checkResourcesModifiedRemotely) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for range l.kinds {
		// TODO: for each kind in the allowlist, list resources of that kind from
		// the terraform state (see ucm/deploy/terraform state parsing), fetch the
		// corresponding live object from the workspace, compare the server-side
		// fingerprint (etag / update_time) against what's recorded in state, and
		// append a diag.Warning per drifted resource. Mirrors
		// bundle/deploy/terraform/check_dashboards_modified_remotely.go but
		// generic over the resource kind. Honour an equivalent of
		// Bundle.Config.Bundle.Force to let users bypass the check.
	}
	return diags
}

// CheckResourcesModifiedRemotely returns a mutator that warns when resources
// of the given kinds have been modified in the workspace since the last
// terraform deploy. The current implementation is a scaffold: with an empty
// kinds slice it is a no-op. Concrete resource checks will be wired in later
// tasks once UCM picks specific UC resource kinds to cover.
func CheckResourcesModifiedRemotely(kinds []string) *checkResourcesModifiedRemotely {
	return &checkResourcesModifiedRemotely{kinds: kinds}
}
