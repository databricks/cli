package cmdctx

import (
	"context"

	"github.com/databricks/cli/libs/sdkclients"
)

func SetWorkspaceClient(ctx context.Context, w *sdkclients.Clients) context.Context {
	if v := ctx.Value(workspaceClientKey); v != nil {
		panic("cmdctx.SetWorkspaceClient called twice on the same context.")
	}
	return context.WithValue(ctx, workspaceClientKey, w)
}

func WorkspaceClient(ctx context.Context) *sdkclients.Clients {
	v := ctx.Value(workspaceClientKey)
	if v == nil {
		panic("cmdctx.WorkspaceClient called without calling cmdctx.SetWorkspaceClient first.")
	}
	return v.(*sdkclients.Clients)
}

// HasWorkspaceClient reports whether a workspace client was configured on the
// context via SetWorkspaceClient.
func HasWorkspaceClient(ctx context.Context) bool {
	return ctx.Value(workspaceClientKey) != nil
}
