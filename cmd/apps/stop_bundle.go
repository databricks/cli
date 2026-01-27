package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

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
							handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, appName, []string{"STOPPED state", "already"}, "stopped", wrapError)
							if handled {
								return handledErr
							}
							return wrapError(cmd, appName, err)
						}

						var appInfo *apps.App
						if shouldWaitForCompletion(cmd) {
							spinner := cmdio.Spinner(ctx)
							appInfo, err = wait.OnProgress(createAppProgressCallback(spinner)).GetWithTimeout(getWaitTimeout(cmd))
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

						message := formatAppStatusMessage(appInfo, appName, "stopped")
						cmdio.LogString(ctx, message)
						return nil
					}

					return originalRunE(cmd, []string{appName})
				}
			}

			err := originalRunE(cmd, args)
			if err != nil {
				handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, stopReq.Name, []string{"STOPPED state", "already"}, "stopped", wrapError)
				if handled {
					return handledErr
				}
			}
			return wrapError(cmd, stopReq.Name, err)
		}

		updateCommandHelp(stopCmd, "Stop", "stop")
	}
}
