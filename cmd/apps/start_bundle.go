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
						if err != nil {
							handled, handledErr := handleAlreadyInStateError(ctx, cmd, err, appName, []string{"ACTIVE state", "already"}, "is deployed", wrapError)
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
