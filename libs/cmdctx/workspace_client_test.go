package cmdctx_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestCommandWorkspaceClient(t *testing.T) {
	ctx := t.Context()
	client := &databricks.WorkspaceClient{
		Config: &config.Config{
			Host: "https://test.test",
		},
	}

	// Panic if WorkspaceClient is called before SetWorkspaceClient.
	assert.Panics(t, func() {
		cmdctx.WorkspaceClient(ctx)
	})

	ctx = cmdctx.SetWorkspaceClient(t.Context(), client)

	// Multiple calls should return a pointer to the same client.
	w := cmdctx.WorkspaceClient(ctx)
	assert.Same(t, w, cmdctx.WorkspaceClient(ctx))

	// The client should have the correct configuration.
	assert.Equal(t, "https://test.test", cmdctx.WorkspaceClient(ctx).Config.Host)

	// Second call should panic.
	assert.Panics(t, func() {
		cmdctx.SetWorkspaceClient(ctx, client)
	})
}

func TestHasWorkspaceClient(t *testing.T) {
	ctx := t.Context()
	assert.False(t, cmdctx.HasWorkspaceClient(ctx))

	client := &databricks.WorkspaceClient{
		Config: &config.Config{
			Host: "https://test.test",
		},
	}
	ctx = cmdctx.SetWorkspaceClient(ctx, client)
	assert.True(t, cmdctx.HasWorkspaceClient(ctx))
}
