package cmdctx

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	if v := ctx.Value(workspaceClientKey); v != nil {
		panic("cmdctx.SetWorkspaceClient called twice on the same context.")
	}
	return context.WithValue(ctx, workspaceClientKey, w)
}

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	v := ctx.Value(workspaceClientKey)
	if v == nil {
		panic("cmdctx.WorkspaceClient called without calling cmdctx.SetWorkspaceClient first.")
	}
	return v.(*databricks.WorkspaceClient)
}
