package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/diag"
)

type initializeCache struct{}

// InitializeCache initializes the bundle cache which can be used to cache API responses.
func InitializeCache() bundle.Mutator {
	return &initializeCache{}
}

func (m *initializeCache) Name() string {
	return "InitializeCache"
}

func (m *initializeCache) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Initialize cache with 30 minute expiry for user information
	b.Cache = cache.NewCache(ctx, "user", 30, &b.Metrics)
	return nil
}
