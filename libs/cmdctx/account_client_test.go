package cmdctx

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestCommandAccountClient(t *testing.T) {
	ctx := context.Background()
	client := &databricks.AccountClient{
		Config: &config.Config{
			AccountID: "test-account",
		},
	}

	// Panic if AccountClient is called before SetAccountClient.
	assert.Panics(t, func() {
		AccountClient(ctx)
	})

	ctx = SetAccountClient(context.Background(), client)

	// Multiple calls should return a pointer to the same client.
	a := AccountClient(ctx)
	assert.Same(t, a, AccountClient(ctx))

	// The client should have the correct configuration.
	assert.Equal(t, "test-account", AccountClient(ctx).Config.AccountID)

	// Second call should panic.
	assert.Panics(t, func() {
		SetAccountClient(ctx, client)
	})
}
