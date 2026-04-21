package deploy

import (
	"context"
	"fmt"
	"path"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/databricks-sdk-go"
)

// DefaultWorkspaceStateRoot is the per-user workspace directory where ucm
// stores remote state when no explicit `ucm.state.backend` is configured.
// The ucm.name and selected target are appended at resolution time so
// multi-project / multi-target users don't collide.
const DefaultWorkspaceStateRoot = "databricks/ucm"

// BackendFromUcm constructs a production Backend from the ucm config and the
// workspace client attached to ctx via cmdctx.SetWorkspaceClient. It resolves
// the workspace state path (v1: `~/<DefaultWorkspaceStateRoot>/<name>/<target>/state`),
// instantiates a workspace-files filer, and wraps it as both StateFiler and
// LockFiler. The pluggable `ucm.state.backend` config selector lands in a
// later milestone; for now only the workspace backend is wired.
func BackendFromUcm(ctx context.Context, u *ucm.Ucm, w *databricks.WorkspaceClient) (Backend, error) {
	if u == nil {
		return Backend{}, fmt.Errorf("ucm deploy: BackendFromUcm called with nil Ucm")
	}
	if w == nil {
		return Backend{}, fmt.Errorf("ucm deploy: BackendFromUcm called with nil workspace client")
	}

	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return Backend{}, fmt.Errorf("ucm deploy: resolve current user: %w", err)
	}
	if me.UserName == "" {
		return Backend{}, fmt.Errorf("ucm deploy: current user has no username")
	}

	name := u.Config.Ucm.Name
	if name == "" {
		return Backend{}, fmt.Errorf("ucm deploy: ucm.name is required to resolve the state path")
	}
	target := u.Config.Ucm.Target
	if target == "" {
		return Backend{}, fmt.Errorf("ucm deploy: no target selected; call LoadDefaultTarget or LoadNamedTarget first")
	}

	root := path.Join("/Users", me.UserName, DefaultWorkspaceStateRoot, name, target, "state")

	inner, err := libsfiler.NewWorkspaceFilesClient(w, root)
	if err != nil {
		return Backend{}, fmt.Errorf("ucm deploy: init workspace-files filer at %s: %w", root, err)
	}

	return Backend{
		StateFiler: ucmfiler.NewStateFilerFromFiler(inner),
		LockFiler:  inner,
		User:       me.UserName,
	}, nil
}
