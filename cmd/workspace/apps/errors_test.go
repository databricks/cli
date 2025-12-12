package apps

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppDeploymentError_Error_WithoutProfile(t *testing.T) {
	originalErr := errors.New("deployment failed: timeout")
	appErr := &AppDeploymentError{
		Underlying: originalErr,
		appName:    "my-app",
		profile:    "",
	}

	result := appErr.Error()

	assert.Contains(t, result, "deployment failed: timeout")
	assert.Contains(t, result, "To view app logs, run:")
	assert.Contains(t, result, "databricks apps logs my-app --tail-lines 100")
	assert.NotContains(t, result, "--profile")
}

func TestAppDeploymentError_Error_WithProfile(t *testing.T) {
	originalErr := errors.New("deployment failed: timeout")
	appErr := &AppDeploymentError{
		Underlying: originalErr,
		appName:    "my-app",
		profile:    "production",
	}

	result := appErr.Error()

	assert.Contains(t, result, "deployment failed: timeout")
	assert.Contains(t, result, "To view app logs, run:")
	assert.Contains(t, result, "databricks apps logs my-app --tail-lines 100 --profile production")
}

func TestAppDeploymentError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := &AppDeploymentError{
		Underlying: originalErr,
		appName:    "my-app",
		profile:    "",
	}

	unwrapped := appErr.Unwrap()

	require.Equal(t, originalErr, unwrapped)
	assert.ErrorIs(t, appErr, originalErr, "errors.Is should work with wrapped error")
}
