package lakebase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "server closed connection unexpectedly",
			output:   "psql: error: connection to server at \"instance-xyz.database.cloud.databricks.com\" (44.234.192.47), port 5432 failed: server closed the connection unexpectedly",
			expected: true,
		},
		{
			name:     "connection to server failed",
			output:   "psql: error: connection to server at \"instance-xyz.database.cloud.databricks.com\" failed",
			expected: true,
		},
		{
			name:     "external authorization failed",
			output:   "psql: error: External authorization failed. DETAIL: This could be due to paused instances",
			expected: true,
		},
		{
			name:     "blocked by IP ACL",
			output:   "psql: error: Source IP address: 108.224.88.11 is blocked by Databricks IP ACL for workspace: 12345678900987654321",
			expected: true,
		},
		{
			name:     "connection refused",
			output:   "psql: error: Connection refused",
			expected: true,
		},
		{
			name:     "connection timeout",
			output:   "psql: error: Connection timed out",
			expected: true,
		},
		{
			name:     "non-retryable error - syntax error",
			output:   "psql: error: syntax error at line 1",
			expected: false,
		},
		{
			name:     "non-retryable error - authentication failed",
			output:   "psql: FATAL: password authentication failed for user",
			expected: false,
		},
		{
			name:     "non-retryable error - database does not exist",
			output:   "psql: FATAL: database \"nonexistent\" does not exist",
			expected: false,
		},
		{
			name:     "empty output",
			output:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableConnectionError([]byte(tt.output))
			assert.Equal(t, tt.expected, result, "Test case: %s", tt.name)
		})
	}
}

func TestRetryConfigDefaults(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    defaultMaxRetries,
		InitialDelay:  defaultInitialDelay,
		MaxDelay:      defaultMaxDelay,
		BackoffFactor: defaultBackoffFactor,
	}

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, "1s", config.InitialDelay.String())
	assert.Equal(t, "10s", config.MaxDelay.String())
	assert.InEpsilon(t, 2.0, config.BackoffFactor, 0.001)
}

func TestTryPsqlInteractive(t *testing.T) {
	ctx := context.Background()

	// Test successful execution (exit code 0)
	args := []string{"echo", "success"}
	var env []string
	err := tryPsqlInteractive(ctx, args, env)
	assert.NoError(t, err)

	// Test connection failure (exit code 2) - simulate with false command
	args = []string{"sh", "-c", "exit 2"}
	err = tryPsqlInteractive(ctx, args, env)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed (retryable)")
	assert.Contains(t, err.Error(), "psql exited with code 2")

	// Test other failure (exit code 1) - should not be retryable
	args = []string{"sh", "-c", "exit 1"}
	err = tryPsqlInteractive(ctx, args, env)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "connection failed (retryable)")
}
