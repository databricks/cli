package apps

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// BundleStartOverrideWithWrapper creates a start override function that uses
// the provided error wrapper for API fallback errors.
func BundleStartOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.StartAppRequest) {
	return func(startCmd *cobra.Command, startReq *apps.StartAppRequest) {
		// Update the command usage to reflect that NAME is optional when in project mode
		startCmd.Use = "start [NAME]"

		// Override Args to allow 0 or 1 arguments (project mode vs API mode)
		startCmd.Args = func(cmd *cobra.Command, args []string) error {
			// Never allow more than 1 argument
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			// In non-project mode, exactly 1 argument is required
			if !hasBundleConfig() && len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			// In project mode: 0 args = use bundle config, 1 arg = API fallback
			return nil
		}

		originalRunE := startCmd.RunE
		startCmd.RunE = func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// If no NAME provided, try to detect from project config
			if len(args) == 0 {
				appName := detectAppNameFromBundle(cmd)
				if appName != "" {
					cmdio.LogString(ctx, fmt.Sprintf("Starting app '%s' from project configuration", appName))
					startReq.Name = appName
					return originalRunE(cmd, []string{appName})
				}
				return errors.New("no app name provided and unable to detect from project configuration")
			}

			// Otherwise, fall back to the original API start command
			err := originalRunE(cmd, args)
			return wrapError(cmd, startReq.Name, err)
		}

		// Update the help text to explain the dual behavior
		startCmd.Long = `Start an app.

When run from a Databricks Apps project directory (containing databricks.yml)
without a NAME argument, this command automatically detects the app name from
the project configuration and starts it.

When a NAME argument is provided (or when not in a project directory),
starts the specified app using the API directly.

Arguments:
  NAME: The name of the app. Required when not in a project directory.
        When provided in a project directory, uses the specified name instead of auto-detection.

Examples:
  # Start app from a project directory (auto-detects app name)
  databricks apps start

  # Start a specific app using the API (even from a project directory)
  databricks apps start my-app`
	}
}

// BundleStopOverrideWithWrapper creates a stop override function that uses
// the provided error wrapper for API fallback errors.
func BundleStopOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.StopAppRequest) {
	return func(stopCmd *cobra.Command, stopReq *apps.StopAppRequest) {
		// Update the command usage to reflect that NAME is optional when in project mode
		stopCmd.Use = "stop [NAME]"

		// Override Args to allow 0 or 1 arguments (project mode vs API mode)
		stopCmd.Args = func(cmd *cobra.Command, args []string) error {
			// Never allow more than 1 argument
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			// In non-project mode, exactly 1 argument is required
			if !hasBundleConfig() && len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			// In project mode: 0 args = use bundle config, 1 arg = API fallback
			return nil
		}

		originalRunE := stopCmd.RunE
		stopCmd.RunE = func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// If no NAME provided, try to detect from project config
			if len(args) == 0 {
				appName := detectAppNameFromBundle(cmd)
				if appName != "" {
					cmdio.LogString(ctx, fmt.Sprintf("Stopping app '%s' from project configuration", appName))
					stopReq.Name = appName
					return originalRunE(cmd, []string{appName})
				}
				return errors.New("no app name provided and unable to detect from project configuration")
			}

			// Otherwise, fall back to the original API stop command
			err := originalRunE(cmd, args)
			return wrapError(cmd, stopReq.Name, err)
		}

		// Update the help text to explain the dual behavior
		stopCmd.Long = `Stop an app.

When run from a Databricks Apps project directory (containing databricks.yml)
without a NAME argument, this command automatically detects the app name from
the project configuration and stops it.

When a NAME argument is provided (or when not in a project directory),
stops the specified app using the API directly.

Arguments:
  NAME: The name of the app. Required when not in a project directory.
        When provided in a project directory, uses the specified name instead of auto-detection.

Examples:
  # Stop app from a project directory (auto-detects app name)
  databricks apps stop

  # Stop a specific app using the API (even from a project directory)
  databricks apps stop my-app`
	}
}
