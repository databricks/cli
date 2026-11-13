package cmdctx

import (
	"context"

	"github.com/databricks/databricks-sdk-go/config"
)

func SetConfigUsed(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configUsedKey, cfg)
}

func ConfigUsed(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configUsedKey).(*config.Config)
	if !ok {
		panic("cannot get *config.Config. Please report it as a bug")
	}
	return cfg
}

// ClearConfigUsed returns a context with no config used, as if SetConfigUsed had not run: the telemetry upload is
// skipped and auth errors are not enriched.
func ClearConfigUsed(ctx context.Context) context.Context {
	return context.WithValue(ctx, configUsedKey, nil)
}

func HasConfigUsed(ctx context.Context) bool {
	_, ok := ctx.Value(configUsedKey).(*config.Config)
	return ok
}
