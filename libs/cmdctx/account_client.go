package cmdctx

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

func SetAccountClient(ctx context.Context, a *databricks.AccountClient) context.Context {
	if v := ctx.Value(accountClientKey); v != nil {
		panic("cmdctx.SetAccountClient called twice on the same context.")
	}
	return context.WithValue(ctx, accountClientKey, a)
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	a, ok := ctx.Value(accountClientKey).(*databricks.AccountClient)
	if !ok {
		panic("cmdctx.AccountClient called without calling cmdctx.SetAccountClient first")
	}
	return a
}
