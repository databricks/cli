package psql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttemptConnection(t *testing.T) {
	ctx := context.Background()

	// Test successful execution (exit code 0)
	args := []string{"echo", "success"}
	var env []string
	err := attemptConnection(ctx, args, env)
	assert.NoError(t, err)

	// Test connection failure (exit code 2) with no specific error - retryable
	args = []string{"sh", "-c", "exit 2"}
	err = attemptConnection(ctx, args, env)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errRetryable)

	// Test connection failure (exit code 2) with non-retryable error
	args = []string{"sh", "-c", `echo 'FATAL: role "user" does not exist' >&2; exit 2`}
	err = attemptConnection(ctx, args, env)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, errRetryable)

	// Test other failure (exit code 1) - should not be retryable
	args = []string{"sh", "-c", "exit 1"}
	err = attemptConnection(ctx, args, env)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, errRetryable)
}
