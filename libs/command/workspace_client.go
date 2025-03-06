package command

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	if v := ctx.Value(workspaceClientKey); v != nil {
		panic("command.SetWorkspaceClient called twice on the same context. Please report this bug.")
	}
	return context.WithValue(ctx, workspaceClientKey, w)
}

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	v := ctx.Value(workspaceClientKey)
	if v == nil {
		panic("command.WorkspaceClient called without calling command.SetWorkspaceClient first. Please report this bug.")
	}
	return v.(*databricks.WorkspaceClient)
}
