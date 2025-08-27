package lakebase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
