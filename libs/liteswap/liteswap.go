package liteswap

import (
	"context"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

// TargetVariable names the liteswap unit to route requests to (dev/test only).
const TargetVariable = "DATABRICKS_LITESWAP_TARGET"

// Apply routes cfg's requests to the liteswap unit named by DATABRICKS_LITESWAP_TARGET.
func Apply(ctx context.Context, cfg *config.Config) {
	target := env.Get(ctx, TargetVariable)
	if target == "" {
		return
	}
	cfg.Headers = config.StaticHeaders(map[string]string{
		"x-databricks-traffic-id": "testenv://liteswap/" + target,
	})
}
