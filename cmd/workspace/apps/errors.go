package apps

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AppDeploymentError wraps deployment errors with a helpful logs command suggestion.
type AppDeploymentError struct {
	Err     error
	AppName string
	Profile string
}

func (e *AppDeploymentError) Error() string {
	suggestion := "\n\nTo view app logs, run:\n  databricks workspace apps logs " + e.AppName
	if e.Profile != "" {
		suggestion = fmt.Sprintf("%s --profile %s", suggestion, e.Profile)
	}
	return e.Err.Error() + suggestion
}

func (e *AppDeploymentError) Unwrap() error {
	return e.Err
}

// newAppDeploymentError creates an AppDeploymentError with profile info from the command.
func newAppDeploymentError(cmd *cobra.Command, appName string, err error) error {
	profile := ""
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Value.String() != "" {
		profile = profileFlag.Value.String()
	}
	return &AppDeploymentError{
		Err:     err,
		AppName: appName,
		Profile: profile,
	}
}
