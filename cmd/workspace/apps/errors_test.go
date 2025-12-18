package apps

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/spf13/cobra"
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

func TestWrapDeploymentError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		appName       string
		expectWrapped bool
		description   string
	}{
		{
			name:          "nil error",
			err:           nil,
			appName:       "test-app",
			expectWrapped: false,
			description:   "nil error should return nil unchanged",
		},
		{
			name:          "plain error without retries wrapper",
			err:           errors.New("some error"),
			appName:       "test-app",
			expectWrapped: false,
			description:   "plain errors should not be wrapped",
		},
		{
			name:          "retries.Err with Halt=false",
			err:           retries.Continues("still in progress"),
			appName:       "test-app",
			expectWrapped: false,
			description:   "transient retries errors should not be wrapped",
		},
		{
			name: "retries.Err with 404 API error (not found)",
			err: retries.Halt(fmt.Errorf("API error: %w", &apierr.APIError{
				StatusCode: 404,
				ErrorCode:  "NOT_FOUND",
				Message:    "App with name test-app does not exist or is deleted.",
			})),
			appName:       "test-app",
			expectWrapped: false,
			description:   "404 not found errors should not be wrapped with logs hint",
		},
		{
			name: "retries.Err with 400 API error (bad request)",
			err: retries.Halt(fmt.Errorf("API error: %w", &apierr.APIError{
				StatusCode: 400,
				ErrorCode:  "BAD_REQUEST",
				Message:    "Invalid request parameters",
			})),
			appName:       "test-app",
			expectWrapped: false,
			description:   "400 bad request errors should not be wrapped with logs hint",
		},
		{
			name: "retries.Err with 403 API error (forbidden)",
			err: retries.Halt(fmt.Errorf("API error: %w", &apierr.APIError{
				StatusCode: 403,
				ErrorCode:  "FORBIDDEN",
				Message:    "Access denied",
			})),
			appName:       "test-app",
			expectWrapped: false,
			description:   "403 forbidden errors should not be wrapped with logs hint",
		},
		{
			name: "retries.Err with 500 API error (server error)",
			err: retries.Halt(fmt.Errorf("API error: %w", &apierr.APIError{
				StatusCode: 500,
				ErrorCode:  "INTERNAL_ERROR",
				Message:    "Internal server error",
			})),
			appName:       "test-app",
			expectWrapped: true,
			description:   "500 server errors during wait should be wrapped with logs hint",
		},
		{
			name:          "retries.Err without API error (deployment failure)",
			err:           retries.Halt(errors.New("failed to reach SUCCEEDED, got FAILED: Error building app")),
			appName:       "test-app",
			expectWrapped: true,
			description:   "deployment failures should be wrapped with logs hint",
		},
		{
			name:          "retries.Err without API error (timeout)",
			err:           retries.Halt(errors.New("timeout waiting for deployment")),
			appName:       "test-app",
			expectWrapped: true,
			description:   "timeout errors should be wrapped with logs hint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			result := wrapDeploymentError(cmd, tt.appName, tt.err)

			if tt.expectWrapped {
				var appErr *AppDeploymentError
				require.ErrorAs(t, result, &appErr, tt.description)
				assert.Equal(t, tt.appName, appErr.appName)
				assert.ErrorIs(t, result, tt.err, "wrapped error should unwrap to original")
				assert.Contains(t, result.Error(), "To view app logs, run:", "should contain logs hint")
			} else {
				assert.Equal(t, tt.err, result, tt.description)
				if tt.err != nil {
					assert.NotContains(t, result.Error(), "To view app logs", "should not contain logs hint")
				}
			}
		})
	}
}
