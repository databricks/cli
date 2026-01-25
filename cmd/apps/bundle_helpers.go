package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

const defaultAppWaitTimeout = 20 * time.Minute

// makeArgsOptionalWithBundle updates a command to allow optional NAME argument
// when running from a bundle directory.
func makeArgsOptionalWithBundle(cmd *cobra.Command, usage string) {
	cmd.Use = usage

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
		}
		if !hasBundleConfig() && len(args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	}
}

// getAppNameFromArgs returns the app name from args or detects it from the bundle.
// Returns (appName, fromBundle, error).
func getAppNameFromArgs(cmd *cobra.Command, args []string) (string, bool, error) {
	if len(args) > 0 {
		return args[0], false, nil
	}

	appName := detectAppNameFromBundle(cmd)
	if appName != "" {
		return appName, true, nil
	}

	return "", false, errors.New("no app name provided and unable to detect from project configuration")
}

// updateCommandHelp updates the help text for a command to explain bundle behavior.
func updateCommandHelp(cmd *cobra.Command, commandVerb, commandName string) {
	cmd.Long = fmt.Sprintf(`%s an app.

When run from a Databricks Apps project directory (containing databricks.yml)
without a NAME argument, this command automatically detects the app name from
the project configuration and %ss it.

When a NAME argument is provided (or when not in a project directory),
%ss the specified app using the API directly.

Arguments:
  NAME: The name of the app. Required when not in a project directory.
        When provided in a project directory, uses the specified name instead of auto-detection.

Examples:
  # %s app from a project directory (auto-detects app name)
  databricks apps %s

  # %s app from a specific target
  databricks apps %s --target prod

  # %s a specific app using the API (even from a project directory)
  databricks apps %s my-app`,
		commandVerb,
		commandName,
		commandName,
		commandVerb,
		commandName,
		commandVerb,
		commandName,
		commandVerb,
		commandName)
}

// BundleLogsOverride creates a logs override function that supports bundle mode.
func BundleLogsOverride(logsCmd *cobra.Command) {
	makeArgsOptionalWithBundle(logsCmd, "logs [NAME]")

	originalRunE := logsCmd.RunE
	logsCmd.RunE = func(cmd *cobra.Command, args []string) error {
		appName, fromBundle, err := getAppNameFromArgs(cmd, args)
		if err != nil {
			return err
		}

		if fromBundle && root.OutputType(cmd) != flags.OutputJSON {
			fmt.Fprintf(cmd.ErrOrStderr(), "Streaming logs for app '%s' from project configuration\n", appName)
		}

		return originalRunE(cmd, []string{appName})
	}

	logsCmd.Long = `Show Databricks app logs.

When run from a Databricks Apps project directory (containing databricks.yml)
without a NAME argument, this command automatically detects the app name from
the project configuration and shows its logs.

When a NAME argument is provided (or when not in a project directory),
shows logs for the specified app using the API directly.

Arguments:
  NAME: The name of the app. Required when not in a project directory.
        When provided in a project directory, uses the specified name instead of auto-detection.

Examples:
  # Show logs from a project directory (auto-detects app name)
  databricks apps logs

  # Show logs from a specific target
  databricks apps logs --target prod

  # Show logs for a specific app using the API (even from a project directory)
  databricks apps logs my-app`
}

// isIdempotencyError checks if an error message indicates the operation is already in the desired state.
func isIdempotencyError(err error, keywords ...string) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

// displayAppURL displays the app URL in a consistent format if available.
func displayAppURL(ctx context.Context, appInfo *apps.App) {
	if appInfo != nil && appInfo.Url != "" {
		cmdio.LogString(ctx, fmt.Sprintf("\n🔗 %s\n", appInfo.Url))
	}
}

// formatAppStatusMessage formats a user-friendly status message for an app.
func formatAppStatusMessage(appInfo *apps.App, appName, verb string) string {
	computeState := "unknown"
	if appInfo != nil && appInfo.ComputeStatus != nil {
		computeState = string(appInfo.ComputeStatus.State)
	}

	if appInfo != nil && appInfo.AppStatus != nil && appInfo.AppStatus.State == apps.ApplicationStateUnavailable {
		return fmt.Sprintf("⚠ App '%s' %s but is unavailable (compute: %s, app: %s)", appName, verb, computeState, appInfo.AppStatus.State)
	}

	if appInfo != nil && appInfo.ComputeStatus != nil {
		state := appInfo.ComputeStatus.State
		switch state {
		case apps.ComputeStateActive:
			if verb == "is deployed" {
				return fmt.Sprintf("✔ App '%s' is already running (status: %s)", appName, state)
			}
			return fmt.Sprintf("✔ App '%s' started successfully (status: %s)", appName, state)
		case apps.ComputeStateStarting:
			return fmt.Sprintf("⚠ App '%s' is already starting (status: %s)", appName, state)
		default:
			return fmt.Sprintf("✔ App '%s' status: %s", appName, state)
		}
	}

	return fmt.Sprintf("✔ App '%s' status: unknown", appName)
}

// getWaitTimeout gets the timeout value for app wait operations.
func getWaitTimeout(cmd *cobra.Command) time.Duration {
	timeout, _ := cmd.Flags().GetDuration("timeout")
	if timeout == 0 {
		timeout = defaultAppWaitTimeout
	}
	return timeout
}

// shouldWaitForCompletion checks if the command should wait for app operation completion.
func shouldWaitForCompletion(cmd *cobra.Command) bool {
	skipWait, _ := cmd.Flags().GetBool("no-wait")
	return !skipWait
}

// createAppProgressCallback creates a progress callback for app operations.
func createAppProgressCallback(spinner chan<- string) func(*apps.App) {
	return func(i *apps.App) {
		if i.ComputeStatus == nil {
			return
		}
		statusMessage := i.ComputeStatus.Message
		if statusMessage == "" {
			statusMessage = fmt.Sprintf("current status: %s", i.ComputeStatus.State)
		}
		spinner <- statusMessage
	}
}

// handleAlreadyInStateError handles idempotency errors and displays appropriate status.
// Returns true if the error was handled (already in desired state), false otherwise.
func handleAlreadyInStateError(ctx context.Context, cmd *cobra.Command, err error, appName string, keywords []string, verb string, wrapError ErrorWrapper) (bool, error) {
	if !isIdempotencyError(err, keywords...) {
		return false, nil
	}

	outputFormat := root.OutputType(cmd)
	if outputFormat != flags.OutputText {
		return true, nil
	}

	w := cmdctx.WorkspaceClient(ctx)
	appInfo, getErr := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
	if getErr != nil {
		return true, wrapError(cmd, appName, getErr)
	}

	message := formatAppStatusMessage(appInfo, appName, verb)
	cmdio.LogString(ctx, message)
	displayAppURL(ctx, appInfo)
	return true, nil
}
