// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_conf

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace-conf",
		Short:   `This API allows updating known workspace settings for advanced users.`,
		Long:    `This API allows updating known workspace settings for advanced users.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetStatus())
	cmd.AddCommand(newSetStatus())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getStatusOverrides []func(
	*cobra.Command,
	*settings.GetStatusRequest,
)

func newGetStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var getStatusReq settings.GetStatusRequest

	cmd.Use = "get-status KEYS"
	cmd.Short = `Check configuration status.`
	cmd.Long = `Check configuration status.
  
  Gets the configuration status for a workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getStatusReq.Keys = args[0]

		response, err := w.WorkspaceConf.GetStatus(ctx, getStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getStatusOverrides {
		fn(cmd, &getStatusReq)
	}

	return cmd
}

// start set-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setStatusOverrides []func(
	*cobra.Command,
	*settings.WorkspaceConf,
)

func newSetStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var setStatusReq settings.WorkspaceConf
	var setStatusJson flags.JsonFlag

	cmd.Flags().Var(&setStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-status"
	cmd.Short = `Enable/disable features.`
	cmd.Long = `Enable/disable features.
  
  Sets the configuration status for a workspace, including enabling or disabling
  it.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setStatusJson.Unmarshal(&setStatusReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.WorkspaceConf.SetStatus(ctx, setStatusReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setStatusOverrides {
		fn(cmd, &setStatusReq)
	}

	return cmd
}

// end service WorkspaceConf
