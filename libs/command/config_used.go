package command

import (
	"context"

	"github.com/databricks/databricks-sdk-go/config"
)

func SetConfigUsed(ctx context.Context, cfg *config.Config) context.Context {
	if v := ctx.Value(configUsedKey); v != nil {
		panic("command.SetConfigUsed called twice on the same context")
	}
	return context.WithValue(ctx, configUsedKey, cfg)
}

func ConfigUsed(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configUsedKey).(*config.Config)
	if !ok {
		panic("cannot get *config.Config. Please report it as a bug")
	}
	return cfg
}
