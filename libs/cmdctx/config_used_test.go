package cmdctx

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestCommandConfigUsed(t *testing.T) {
	cfg := &config.Config{
		Host: "https://test.test",
	}
	ctx := t.Context()

	// Panic if ConfigUsed is called before SetConfigUsed.
	assert.Panics(t, func() {
		ConfigUsed(ctx)
	})

	ctx = SetConfigUsed(ctx, cfg)

	// Multiple calls should return a pointer to the same config.
	c := ConfigUsed(ctx)
	assert.Same(t, c, ConfigUsed(ctx))

	// The config should have the correct configuration.
	assert.Equal(t, "https://test.test", ConfigUsed(ctx).Host)

	// Second call should update the config used.
	cfg2 := &config.Config{
		Host: "https://test2.test",
	}
	ctx = SetConfigUsed(ctx, cfg2)
	assert.Equal(t, "https://test2.test", ConfigUsed(ctx).Host)
}
