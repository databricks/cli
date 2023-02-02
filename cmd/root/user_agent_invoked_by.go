package root

import (
	"context"
	"os"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// Environment variable that caller can set to convey what invoked bricks.
const invokedByEnvironmentVariable = "BRICKS_INVOKED_BY"

// Key in the user agent.
const invokedByKey = "invoked-by"

func withInvokedByInUserAgent(ctx context.Context) context.Context {
	value, ok := os.LookupEnv(invokedByEnvironmentVariable)
	if !ok {
		return ctx
	}
	return useragent.InContext(ctx, invokedByKey, value)
}
