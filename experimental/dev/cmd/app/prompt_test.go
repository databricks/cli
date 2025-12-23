package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple name",
			projectName: "my-app",
			expectError: false,
		},
		{
			name:        "valid name with numbers",
			projectName: "app123",
			expectError: false,
		},
		{
			name:        "valid name with hyphens",
			projectName: "my-cool-app",
			expectError: false,
		},
		{
			name:        "empty name",
			projectName: "",
			expectError: true,
			errorMsg:    "required",
		},
		{
			name:        "name too long",
			projectName: "this-is-a-very-long-app-name-that-exceeds",
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "name at max length (26 chars)",
			projectName: "abcdefghijklmnopqrstuvwxyz",
			expectError: false,
		},
		{
			name:        "name starts with number",
			projectName: "123app",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name starts with hyphen",
			projectName: "-myapp",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name with uppercase",
			projectName: "MyApp",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "name with underscore",
			projectName: "my_app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with spaces",
			projectName: "my app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with special characters",
			projectName: "my@app!",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.projectName)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunWithSpinnerCtx(t *testing.T) {
	t.Run("successful action", func(t *testing.T) {
		ctx := context.Background()
		executed := false

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("action returns error", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("action failed")

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		actionStarted := make(chan struct{})
		actionDone := make(chan struct{})

		go func() {
			_ = RunWithSpinnerCtx(ctx, "Testing...", func() error {
				close(actionStarted)
				time.Sleep(100 * time.Millisecond)
				close(actionDone)
				return nil
			})
		}()

		// Wait for action to start
		<-actionStarted
		// Cancel context
		cancel()
		// Wait for action to complete (spinner should wait)
		<-actionDone
	})

	t.Run("action panics - recovered", func(t *testing.T) {
		ctx := context.Background()

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			panic("test panic")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "action panicked")
		assert.Contains(t, err.Error(), "test panic")
	})
}

func TestRunModeConstants(t *testing.T) {
	assert.Equal(t, RunMode("none"), RunModeNone)
	assert.Equal(t, RunMode("dev"), RunModeDev)
	assert.Equal(t, RunMode("dev-remote"), RunModeDevRemote)
}

func TestMaxAppNameLength(t *testing.T) {
	// Verify the constant is set correctly
	assert.Equal(t, 30, MaxAppNameLength)
	assert.Equal(t, "dev-", DevTargetPrefix)

	// Max allowed name length should be 30 - 4 ("dev-") = 26
	maxAllowed := MaxAppNameLength - len(DevTargetPrefix)
	assert.Equal(t, 26, maxAllowed)

	// Test at boundary
	validName := "abcdefghijklmnopqrstuvwxyz" // 26 chars
	assert.Len(t, validName, 26)
	assert.NoError(t, ValidateProjectName(validName))

	// Test over boundary
	invalidName := "abcdefghijklmnopqrstuvwxyz1" // 27 chars
	assert.Len(t, invalidName, 27)
	assert.Error(t, ValidateProjectName(invalidName))
}
