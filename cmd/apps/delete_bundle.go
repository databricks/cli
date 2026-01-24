package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// BundleDeleteOverrideWithWrapper creates a delete override function that uses
// the provided error wrapper for API fallback errors.
func BundleDeleteOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.DeleteAppRequest) {
	return func(deleteCmd *cobra.Command, deleteReq *apps.DeleteAppRequest) {
		var (
			autoApprove  bool
			forceDestroy bool
		)

		deleteCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources and files")
		deleteCmd.Flags().BoolVar(&forceDestroy, "force-lock", false, "Force acquisition of deployment lock.")

		// Update the command usage to reflect that APP_NAME is optional when in bundle mode
		deleteCmd.Use = "delete [NAME]"

		// Override Args to allow 0 or 1 arguments (project mode vs API mode)
		deleteCmd.Args = func(cmd *cobra.Command, args []string) error {
			// Never allow more than 1 argument
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			// In non-project mode, exactly 1 argument is required
			if !hasBundleConfig() && len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			// In project mode: 0 args = destroy project, 1 arg = API fallback
			return nil
		}

		originalRunE := deleteCmd.RunE
		deleteCmd.RunE = func(cmd *cobra.Command, args []string) error {
			// If no APP_NAME provided, try to use project destroy flow
			if len(args) == 0 && hasBundleConfig() {
				return bundle.CommandBundleDestroy(cmd, args, autoApprove, forceDestroy)
			}

			// Otherwise, fall back to the original API delete command
			err := originalRunE(cmd, args)
			return wrapError(cmd, deleteReq.Name, err)
		}

		// Update the help text to explain the dual behavior
		deleteCmd.Long = `Delete an app.

When run from a Databricks Apps project directory (containing databricks.yml)
without a NAME argument, this command destroys all resources deployed by the project.

When a NAME argument is provided (or when not in a project directory),
deletes the specified app using the API directly.

Arguments:
  NAME: The name of the app. Required when not in a project directory.
        When provided in a project directory, uses API delete instead of project destroy.

Examples:
  # Destroy all project resources from a project directory
  databricks apps delete

  # Destroy project resources with auto-approval
  databricks apps delete --auto-approve

  # Delete a specific app resource using the API (even from a project directory)
  databricks apps delete my-app-resource`
	}
}
