package apps

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/sdk-go/core/apierr"
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
			name:          "error outside of the wait",
			err:           errors.New("still in progress"),
			appName:       "test-app",
			expectWrapped: false,
			description:   "transient retries errors should not be wrapped",
		},
		{
			name:          "wait error with 404 API error (not found)",
			err:           waitErr(fmt.Errorf("API error: %w", apierr.FromHTTPError(404, nil, []byte(`{"error_code":"NOT_FOUND","message":"App with name test-app does not exist or is deleted."}`)))),
			appName:       "test-app",
			expectWrapped: false,
			description:   "404 not found errors should not be wrapped with logs hint",
		},
		{
			name:          "wait error with 400 API error (bad request)",
			err:           waitErr(fmt.Errorf("API error: %w", apierr.FromHTTPError(400, nil, []byte(`{"error_code":"BAD_REQUEST","message":"Invalid request parameters"}`)))),
			appName:       "test-app",
			expectWrapped: false,
			description:   "400 bad request errors should not be wrapped with logs hint",
		},
		{
			name:          "wait error with 403 API error (forbidden)",
			err:           waitErr(fmt.Errorf("API error: %w", apierr.FromHTTPError(403, nil, []byte(`{"error_code":"FORBIDDEN","message":"Access denied"}`)))),
			appName:       "test-app",
			expectWrapped: false,
			description:   "403 forbidden errors should not be wrapped with logs hint",
		},
		{
			name:          "wait error with 500 API error (server error)",
			err:           waitErr(fmt.Errorf("API error: %w", apierr.FromHTTPError(500, nil, []byte(`{"error_code":"INTERNAL_ERROR","message":"Internal server error"}`)))),
			appName:       "test-app",
			expectWrapped: true,
			description:   "500 server errors during wait should be wrapped with logs hint",
		},
		{
			name:          "wait error without API error (deployment failure)",
			err:           waitErr(errors.New("failed to reach SUCCEEDED, got FAILED: Error building app")),
			appName:       "test-app",
			expectWrapped: true,
			description:   "deployment failures should be wrapped with logs hint",
		},
		{
			name:          "wait error without API error (timeout)",
			err:           waitErr(errors.New("timeout waiting for deployment")),
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

func waitErr(err error) error {
	return &cmdgen.WaitError{Err: err}
}
