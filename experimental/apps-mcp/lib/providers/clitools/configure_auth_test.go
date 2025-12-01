package clitools

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureAuthWithSkipCheck(t *testing.T) {
	// Set skip auth check for testing
	os.Setenv("DATABRICKS_MCP_SKIP_AUTH_CHECK", "1")
	defer os.Unsetenv("DATABRICKS_MCP_SKIP_AUTH_CHECK")

	ctx := context.Background()
	sess := session.NewSession()

	host := "https://test.cloud.databricks.com"
	profile := "test-profile"

	client, err := ConfigureAuth(ctx, sess, &host, &profile)
	require.NoError(t, err)
	assert.Nil(t, client) // Should be nil when skip check is enabled

	// Verify nothing was stored in session when skip check is on
	_, ok := sess.Get(middlewares.DatabricksClientKey)
	assert.False(t, ok)
}

func TestConfigureAuthStoresClientInSession(t *testing.T) {
	// This test requires a valid Databricks configuration
	// Skip if no config is available
	if os.Getenv("DATABRICKS_HOST") == "" && os.Getenv("DATABRICKS_PROFILE") == "" {
		t.Skip("Skipping test: no Databricks configuration found")
	}

	ctx := context.Background()
	sess := session.NewSession()

	client, err := ConfigureAuth(ctx, sess, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify client was stored in session
	stored, ok := sess.Get(middlewares.DatabricksClientKey)
	assert.True(t, ok)
	assert.Equal(t, client, stored)
}

func TestConfigureAuthWithCustomHost(t *testing.T) {
	// This test requires valid credentials
	// Skip if no config is available
	if os.Getenv("DATABRICKS_HOST") == "" {
		t.Skip("Skipping test: DATABRICKS_HOST not set")
	}

	ctx := context.Background()
	sess := session.NewSession()

	host := os.Getenv("DATABRICKS_HOST")
	client, err := ConfigureAuth(ctx, sess, &host, nil)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify the host was set correctly
	assert.Equal(t, host, client.Config.Host)

	// Verify client was stored in session
	_, ok := sess.Get(middlewares.DatabricksClientKey)
	assert.True(t, ok)
}

func TestWrapAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "regular error",
			err:      assert.AnError,
			expected: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := wrapAuthError(tt.err)
			assert.Contains(t, wrapped.Error(), tt.expected)
		})
	}
}
