package env

import (
	"context"

	"github.com/databricks/databricks-sdk-go/config"
)

// NewConfigLoader creates Databricks SDK Config loader that is aware of env.Set variables:
//
//	ctx = env.Set(ctx, "DATABRICKS_WAREHOUSE_ID", "...")
//
// Usage:
//
//	   &config.Config{
//			Loaders:    []config.Loader{
//				env.NewConfigLoader(ctx),
//				config.ConfigAttributes,
//				config.ConfigFile,
//			},
//		}
func NewConfigLoader(ctx context.Context) *configLoader {
	return &configLoader{
		ctx: ctx,
	}
}

type configLoader struct {
	ctx context.Context
}

func (le *configLoader) Name() string {
	return "cli-env"
}

func (le *configLoader) Configure(cfg *config.Config) error {
	for _, a := range config.ConfigAttributes {
		if !a.IsZero(cfg) {
			continue
		}
		for _, k := range a.EnvVars {
			v := Get(le.ctx, k)
			if v == "" {
				continue
			}
			if err := a.Set(cfg, v); err != nil {
				return err
			}
		}
	}
	return nil
}
