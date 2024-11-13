package apps

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// We override apps.Update command beccause currently genkit does not support
// a way to identify that path field (such as name) matches the field in the request body.
// As a result, genkit generates a command with 2 required same fields, update NAME NAME.
// This override should be removed when genkit supports this.
func updateOverride(cmd *cobra.Command, req *apps.UpdateAppRequest) {
	cmd.Use = "update NAME"
	cmd.Long = `Update an app.

  Updates the app with the supplied name.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens. It must be unique within the workspace.`

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	updateJson := cmd.Flag("json").Value.(*flags.JsonFlag)
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&req.App)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		req.Name = args[0]
		response, err := w.Apps.Update(ctx, *req)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}
}

func init() {
	updateOverrides = append(updateOverrides, updateOverride)
}
