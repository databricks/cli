// Copied from cmd/root/user_agent_upstream.go and adapted for pipelines use.
package root

import (
	"context"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// Environment variables that caller can set to convey what is upstream to this CLI.
const (
	upstreamEnvVar        = "DATABRICKS_CLI_UPSTREAM"
	upstreamVersionEnvVar = "DATABRICKS_CLI_UPSTREAM_VERSION"
)

// Keys in the user agent.
const (
	upstreamKey        = "upstream"
	upstreamVersionKey = "upstream-version"
)

func withUpstreamInUserAgent(ctx context.Context) context.Context {
	value := env.Get(ctx, upstreamEnvVar)
	if value == "" {
		return ctx
	}

	ctx = useragent.InContext(ctx, upstreamKey, value)

	// Include upstream version as well, if set.
	value = env.Get(ctx, upstreamVersionEnvVar)
	if value == "" {
		return ctx
	}

	return useragent.InContext(ctx, upstreamVersionKey, value)
}
