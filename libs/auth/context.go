package auth

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
)

// Placeholders to use as unique keys in context.Context.
var (
	workspaceClient int
	accountClient   int
	configUsed      int
)

func WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	w, ok := ctx.Value(&workspaceClient).(*databricks.WorkspaceClient)
	if !ok {
		panic("cannot get *databricks.WorkspaceClient. Please report it as a bug")
	}
	return w
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	a, ok := ctx.Value(&accountClient).(*databricks.AccountClient)
	if !ok {
		panic("cannot get *databricks.AccountClient. Please report it as a bug")
	}
	return a
}

func ConfigUsed(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(&configUsed).(*config.Config)
	if !ok {
		panic("cannot get *config.Config. Please report it as a bug")
	}
	return cfg
}

func SetWorkspaceClient(ctx context.Context, w *databricks.WorkspaceClient) context.Context {
	return context.WithValue(ctx, &workspaceClient, w)
}

func SetAccountClient(ctx context.Context, a *databricks.AccountClient) context.Context {
	return context.WithValue(ctx, &accountClient, a)
}

func SetConfigUsed(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, &configUsed, cfg)
}
