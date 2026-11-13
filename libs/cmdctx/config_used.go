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

func HasConfigUsed(ctx context.Context) bool {
	_, ok := ctx.Value(configUsedKey).(*config.Config)
	return ok
}
