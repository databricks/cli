package apps

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/spf13/cobra"
)

const tailLinesSuggestedValue = 100

// isDeploymentWaitError checks if the error is from the deployment wait phase.
// The generated commands mark errors of the wait as cmdgen.WaitError.
// Excludes API client errors (4xx) which are validation errors before deployment starts.
func isDeploymentWaitError(err error) bool {
	if _, ok := errors.AsType[*cmdgen.WaitError](err); !ok {
		return false
	}

	// Exclude API client errors (4xx) (e.g. app not found)
	if code := apierrs.StatusCode(err); code >= 400 && code < 500 {
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
