package apps

import (
	"fmt"

	"github.com/spf13/cobra"
)

const tailLinesSuggestedValue = 100

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
