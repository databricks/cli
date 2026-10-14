package cmdctx

import (
	"context"

	"github.com/databricks/cli/libs/sdkclients"
)

func SetAccountClient(ctx context.Context, a *sdkclients.Clients) context.Context {
	if v := ctx.Value(accountClientKey); v != nil {
		panic("cmdctx.SetAccountClient called twice on the same context.")
	}
	return context.WithValue(ctx, accountClientKey, a)
}

func AccountClient(ctx context.Context) *sdkclients.Clients {
	a, ok := ctx.Value(accountClientKey).(*sdkclients.Clients)
	if !ok {
		panic("cmdctx.AccountClient called without calling cmdctx.SetAccountClient first")
	}
	return a
}
