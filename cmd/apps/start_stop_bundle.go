package apps

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

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

// BundleStartOverrideWithWrapper creates a start override function that uses
// the provided error wrapper for API fallback errors.
func BundleStartOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.StartAppRequest) {
	return func(startCmd *cobra.Command, startReq *apps.StartAppRequest) {
		makeArgsOptionalWithBundle(startCmd, "start [NAME]")

		originalRunE := startCmd.RunE
		startCmd.RunE = func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			outputFormat := root.OutputType(cmd)

			if len(args) == 0 {
				appName, fromBundle, err := getAppNameFromArgs(cmd, args)
				if err != nil {
					return err
				}
				if fromBundle {
					startReq.Name = appName

					if outputFormat == flags.OutputText {
						cmdio.LogString(ctx, fmt.Sprintf("Starting app '%s' from project configuration", appName))

						w := cmdctx.WorkspaceClient(ctx)
						wait, err := w.Apps.Start(ctx, *startReq)

						var appInfo *apps.App
						if err != nil {
							handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, appName, []string{"ACTIVE state", "already"}, "is deployed", wrapError)
							if handled {
								return handledErr
							}
							return wrapError(cmd, appName, err)
						}

						skipWait, _ := cmd.Flags().GetBool("no-wait")
						timeout, _ := cmd.Flags().GetDuration("timeout")
						if timeout == 0 {
							timeout = 20 * time.Minute
						}

						if !skipWait {
							spinner := cmdio.Spinner(ctx)
							appInfo, err = wait.OnProgress(func(i *apps.App) {
								if i.ComputeStatus == nil {
									return
								}
								statusMessage := i.ComputeStatus.Message
								if statusMessage == "" {
									statusMessage = fmt.Sprintf("current status: %s", i.ComputeStatus.State)
								}
								spinner <- statusMessage
							}).GetWithTimeout(timeout)
							close(spinner)
							if err != nil {
								return wrapError(cmd, appName, err)
							}
						} else {
							appInfo, err = w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
							if err != nil {
								return wrapError(cmd, appName, err)
							}
						}

						message := formatAppStatusMessage(appInfo, appName, "started")
						cmdio.LogString(ctx, message)
						displayAppURL(ctx, appInfo)
						return nil
					}

					return originalRunE(cmd, []string{appName})
				}
			}

			err := originalRunE(cmd, args)
			if err != nil {
				handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, startReq.Name, []string{"ACTIVE state", "already"}, "is deployed", wrapError)
				if handled {
					return handledErr
				}
			}
			return wrapError(cmd, startReq.Name, err)
		}

		updateCommandHelp(startCmd, "Start", "start")
	}
}

// BundleStopOverrideWithWrapper creates a stop override function that uses
// the provided error wrapper for API fallback errors.
func BundleStopOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.StopAppRequest) {
	return func(stopCmd *cobra.Command, stopReq *apps.StopAppRequest) {
		makeArgsOptionalWithBundle(stopCmd, "stop [NAME]")

		originalRunE := stopCmd.RunE
		stopCmd.RunE = func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			outputFormat := root.OutputType(cmd)

			if len(args) == 0 {
				appName, fromBundle, err := getAppNameFromArgs(cmd, args)
				if err != nil {
					return err
				}
				if fromBundle {
					stopReq.Name = appName

					if outputFormat == flags.OutputText {
						cmdio.LogString(ctx, fmt.Sprintf("Stopping app '%s' from project configuration", appName))

						w := cmdctx.WorkspaceClient(ctx)
						wait, err := w.Apps.Stop(ctx, *stopReq)
						if err != nil {
							if isIdempotencyError(err, "STOPPED state", "already") {
								cmdio.LogString(ctx, fmt.Sprintf("✔ App '%s' is already stopped", appName))
								return nil
							}
							return wrapError(cmd, appName, err)
						}

						skipWait, _ := cmd.Flags().GetBool("no-wait")
						timeout, _ := cmd.Flags().GetDuration("timeout")
						if timeout == 0 {
							timeout = 20 * time.Minute
						}

						if !skipWait {
							spinner := cmdio.Spinner(ctx)
							_, err = wait.OnProgress(func(i *apps.App) {
								if i.ComputeStatus == nil {
									return
								}
								statusMessage := i.ComputeStatus.Message
								if statusMessage == "" {
									statusMessage = fmt.Sprintf("current status: %s", i.ComputeStatus.State)
								}
								spinner <- statusMessage
							}).GetWithTimeout(timeout)
							close(spinner)
							if err != nil {
								return wrapError(cmd, appName, err)
							}
						}

						cmdio.LogString(ctx, fmt.Sprintf("✔ App '%s' stopped successfully", appName))
						return nil
					}

					return originalRunE(cmd, []string{appName})
				}
			}

			err := originalRunE(cmd, args)
			if err != nil {
				if isIdempotencyError(err, "STOPPED state", "already") {
					if outputFormat == flags.OutputText {
						cmdio.LogString(cmd.Context(), fmt.Sprintf("✔ App '%s' is already stopped", stopReq.Name))
					}
					return nil
				}
			}
			return wrapError(cmd, stopReq.Name, err)
		}

		updateCommandHelp(stopCmd, "Stop", "stop")
	}
}
