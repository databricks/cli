package command

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

// func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
// 	if v := ctx.Value(workspaceClientKey); v != nil {
// 		panic("command.SetWorkspaceClient called twice on the same context.")
// 	}
// 	return context.WithValue(ctx, workspaceClientKey, w)
// }

// func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
// 	v := ctx.Value(workspaceClientKey)
// 	if v == nil {
// 		panic("command.WorkspaceClient called without calling command.SetWorkspaceClient first.")
// 	}
// 	return v.(*databricks.WorkspaceClient)
// }

func SetAccountClient(ctx context.Context, a *databricks.AccountClient) context.Context {
	if v := ctx.Value(accountClientKey); v != nil {
		panic("command.SetAccountClient called twice on the same context")
	}
	return context.WithValue(ctx, accountClientKey, a)
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	a, ok := ctx.Value(accountClientKey).(*databricks.AccountClient)
	if !ok {
		panic("command.AccountClient called without calling command.SetAccountClient first")
	}
	return a
}
