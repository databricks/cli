package env

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestLoader(t *testing.T) {
	ctx := context.Background()
	ctx = Set(ctx, "DATABRICKS_WAREHOUSE_ID", "...")
	ctx = Set(ctx, "DATABRICKS_CONFIG_PROFILE", "...")
	loader := NewConfigLoader(ctx)

	cfg := &config.Config{
		Profile: "abc",
	}
	err := loader.Configure(cfg)
	assert.NoError(t, err)

	assert.Equal(t, "...", cfg.WarehouseID)
	assert.Equal(t, "abc", cfg.Profile)
	assert.Equal(t, "cli-env", loader.Name())
}
