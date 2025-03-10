package cmdctx_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestCommandWorkspaceClient(t *testing.T) {
	ctx := context.Background()
	client := &databricks.WorkspaceClient{
		Config: &config.Config{
			Host: "https://test.com",
		},
	}

	// Panic if WorkspaceClient is called before SetWorkspaceClient.
	assert.Panics(t, func() {
		cmdctx.WorkspaceClient(ctx)
	})

	ctx = cmdctx.SetWorkspaceClient(context.Background(), client)

	// Multiple calls should return a pointer to the same client.
	w := cmdctx.WorkspaceClient(ctx)
	assert.Same(t, w, cmdctx.WorkspaceClient(ctx))

	// The client should have the correct configuration.
	assert.Equal(t, "https://test.com", cmdctx.WorkspaceClient(ctx).Config.Host)

	// Second call should panic.
	assert.Panics(t, func() {
		cmdctx.SetWorkspaceClient(ctx, client)
	})
}
