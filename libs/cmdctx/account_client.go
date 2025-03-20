package cmdctx

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

func SetAccountClient(ctx context.Context, a *databricks.AccountClient) context.Context {
	return context.WithValue(ctx, accountClientKey, a)
}

func AccountClient(ctx context.Context) *databricks.AccountClient {
	a, ok := ctx.Value(accountClientKey).(*databricks.AccountClient)
	if !ok {
		panic("command.AccountClient called without calling command.SetAccountClient first")
	}
	return a
}
