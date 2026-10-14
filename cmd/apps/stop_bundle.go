package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
	"github.com/spf13/cobra"
)

// BundleStopOverrideWithWrapper creates a stop override function that uses
// the provided error wrapper for API fallback errors.
func BundleStopOverrideWithWrapper[R any](wrapError ErrorWrapper) func(*cobra.Command, *R) {
	return func(stopCmd *cobra.Command, _ *R) {
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
					if outputFormat == flags.OutputText {
						cmdio.LogString(ctx, fmt.Sprintf("Stopping app '%s' from project configuration", appName))

						w := cmdctx.WorkspaceClient(ctx)
						_, err := w.Apps.StopApp(ctx, sdkapps.StopAppRequest{Name: new(appName)})
						if err != nil {
							handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, appName, []string{"STOPPED state", "already"}, "stopped", wrapError)
							if handled {
								return handledErr
							}
							return wrapError(cmd, appName, err)
						}

						var appInfo *sdkapps.App
						if shouldWaitForCompletion(cmd) {
							spinner := cmdio.NewSpinner(ctx)
							appInfo, err = waitForAppCompute(ctx, w, appName, sdkapps.ComputeStatus_ComputeState_Stopped, getWaitTimeout(cmd), createAppProgressCallback(spinner))
							spinner.Close()
							if err != nil {
								return wrapError(cmd, appName, err)
							}
						} else {
							appInfo, err = w.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(appName)})
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
				handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, appNameArg(args), []string{"STOPPED state", "already"}, "stopped", wrapError)
				if handled {
					return handledErr
				}
			}
			return wrapError(cmd, appNameArg(args), err)
		}

		updateCommandHelp(stopCmd, "Stop", "stop")
	}
}
