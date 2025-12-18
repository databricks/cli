package apps

import (
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/spf13/cobra"
)

const tailLinesSuggestedValue = 100

// isDeploymentWaitError checks if the error is from the deployment wait phase.
// These are errors wrapped by retries.Halt() during GetWithTimeout().
// Excludes API client errors (4xx) which are validation errors before deployment starts.
func isDeploymentWaitError(err error) bool {
	var retriesErr *retries.Err
	if !errors.As(err, &retriesErr) || !retriesErr.Halt {
		return false
	}

	// Exclude API client errors (4xx) (e.g. app not found)
	var apiErr *apierr.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		return false
	}

	return true
}

// wrapDeploymentError wraps the error with logs hint if it's a deployment wait error.
// Returns the original error unchanged if it's not a deployment wait error.
func wrapDeploymentError(cmd *cobra.Command, appName string, err error) error {
	if err != nil && isDeploymentWaitError(err) {
		return newAppDeploymentError(cmd, appName, err)
	}
	return err
}

// AppDeploymentError wraps deployment errors with a helpful logs command suggestion.
type AppDeploymentError struct {
	Underlying error
	appName    string
	profile    string
}

func (e *AppDeploymentError) Error() string {
	suggestion := fmt.Sprintf("\n\nTo view app logs, run:\n  databricks apps logs %s --tail-lines %d",
		e.appName,
		tailLinesSuggestedValue,
	)
	if e.profile != "" {
		suggestion = fmt.Sprintf("%s --profile %s", suggestion, e.profile)
	}
	return e.Underlying.Error() + suggestion
}

func (e *AppDeploymentError) Unwrap() error {
	return e.Underlying
}

// newAppDeploymentError creates an AppDeploymentError with profile info from the command.
func newAppDeploymentError(cmd *cobra.Command, appName string, err error) error {
	profile := ""
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Value.String() != "" {
		profile = profileFlag.Value.String()
	}
	return &AppDeploymentError{
		Underlying: err,
		appName:    appName,
		profile:    profile,
	}
}
