package command_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/command"
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
		command.WorkspaceClient(ctx)
	})

	ctx = command.SetWorkspaceClient(context.Background(), client)

	// Multiple calls should return a pointer to the same client.
	w := command.WorkspaceClient(ctx)
	assert.Same(t, w, command.WorkspaceClient(ctx))

	// The client should have the correct configuration.
	assert.Equal(t, "https://test.com", command.WorkspaceClient(ctx).Config.Host)

	// Second call should panic.
	assert.Panics(t, func() {
		command.SetWorkspaceClient(ctx, client)
	})
}
