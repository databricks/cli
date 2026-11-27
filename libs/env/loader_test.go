package env_test

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestLoader(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "DATABRICKS_WAREHOUSE_ID", "...")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "...")
	loader := env.NewConfigLoader(ctx)

	cfg := &config.Config{
		Profile: "abc",
	}
	err := loader.Configure(cfg)
	assert.NoError(t, err)

	assert.Equal(t, "...", cfg.WarehouseID)
	assert.Equal(t, "abc", cfg.Profile)
	assert.Equal(t, "cli-env", loader.Name())
}

func TestLoaderParsesNonStringAttributes(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "DATABRICKS_DEBUG_HEADERS", "true")
	ctx = env.Set(ctx, "DATABRICKS_RATE_LIMIT", "42")

	cfg := &config.Config{}
	err := env.NewConfigLoader(ctx).Configure(cfg)
	assert.NoError(t, err)

	assert.True(t, cfg.DebugHeaders)
	assert.Equal(t, 42, cfg.RateLimitPerSecond)
}

func TestLoaderReturnsErrorForInvalidValue(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_RATE_LIMIT", "fast")
	err := env.NewConfigLoader(ctx).Configure(&config.Config{})
	assert.ErrorContains(t, err, "DATABRICKS_RATE_LIMIT")
}
