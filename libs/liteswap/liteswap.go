package liteswap

import (
	"context"
	"net/http"

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
	traffic := config.StaticHeaders(map[string]string{
		"x-databricks-traffic-id": "testenv://liteswap/" + target,
	})
	// Keep any header hook already set on cfg, such as the skills-invoked one.
	prev := cfg.Headers
	cfg.Headers = func(r *http.Request) error {
		if prev != nil {
			if err := prev(r); err != nil {
				return err
			}
		}
		return traffic(r)
	}
}
