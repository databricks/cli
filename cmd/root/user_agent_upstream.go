package root

import (
	"context"
	"os"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// Environment variables that caller can set to convey what is upstream to bricks.
const upstreamEnvVar = "BRICKS_UPSTREAM"
const upstreamVersionEnvVar = "BRICKS_UPSTREAM_VERSION"

// Keys in the user agent.
const upstreamKey = "upstream"
const upstreamVersionKey = "upstream-version"

func withUpstreamInUserAgent(ctx context.Context) context.Context {
	value := os.Getenv(upstreamEnvVar)
	if value == "" {
		return ctx
	}

	ctx = useragent.InContext(ctx, upstreamKey, value)

	// Include upstream version as well, if set.
	value = os.Getenv(upstreamVersionEnvVar)
	if value == "" {
		return ctx
	}

	return useragent.InContext(ctx, upstreamVersionKey, value)
}
