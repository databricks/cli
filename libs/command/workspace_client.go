package command

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	if v := ctx.Value(workspaceClientKey); v != nil {
		panic("command.SetWorkspaceClient called twice on the same context.")
	}
	ctx = context.WithValue(ctx, workspaceClientKey, w)

	client, err := client.New(w.Config)
	if err != nil {
		panic(err)
	}
	ctx = context.WithValue(ctx, databricksClientKey, client)
	return ctx
}

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	v := ctx.Value(workspaceClientKey)
	if v == nil {
		panic("command.WorkspaceClient called without calling command.SetWorkspaceClient first.")
	}
	return v.(*databricks.WorkspaceClient)
}

func DatabricksClient(ctx context.Context) *client.DatabricksClient {
	v := ctx.Value(databricksClientKey)
	if v == nil {
		panic("command.DatabricksClient called without calling command.SetWorkspaceClient first.")
	}
	return v.(*client.DatabricksClient)
}
