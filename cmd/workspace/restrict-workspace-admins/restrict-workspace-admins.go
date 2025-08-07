// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package restrict_workspace_admins

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
		Use:   "restrict-workspace-admins",
		Short: `The Restrict Workspace Admins setting lets you control the capabilities of workspace admins.`,
		Long: `The Restrict Workspace Admins setting lets you control the capabilities of
  workspace admins. With the setting status set to ALLOW_ALL, workspace admins
  can create service principal personal access tokens on behalf of any service
  principal in their workspace. Workspace admins can also change a job owner to
  any user in their workspace. And they can change the job run_as setting to any
  user in their workspace or to a service principal on which they have the
  Service Principal User role. With the setting status set to
  RESTRICT_TOKENS_AND_JOB_RUN_AS, workspace admins can only create personal
  access tokens on behalf of service principals they have the Service Principal
  User role on. They can also only change a job owner to themselves. And they
  can change the job run_as setting to themselves or to a service principal on
  which they have the Service Principal User role.`,
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*settings.DeleteRestrictWorkspaceAdminsSettingRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq settings.DeleteRestrictWorkspaceAdminsSettingRequest

	cmd.Flags().StringVar(&deleteReq.Etag, "etag", deleteReq.Etag, `etag used for versioning.`)

	cmd.Use = "delete"
	cmd.Short = `Delete the restrict workspace admins setting.`
	cmd.Long = `Delete the restrict workspace admins setting.
  
  Reverts the restrict workspace admins setting status for the workspace. A
  fresh etag needs to be provided in DELETE requests (as a query parameter).
  The etag can be retrieved by making a GET request before the DELETE request.
  If the setting is updated/deleted concurrently, DELETE fails with 409 and
  the request must be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Settings.RestrictWorkspaceAdmins().Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*settings.GetRestrictWorkspaceAdminsSettingRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetRestrictWorkspaceAdminsSettingRequest

	cmd.Flags().StringVar(&getReq.Etag, "etag", getReq.Etag, `etag used for versioning.`)

	cmd.Use = "get"
	cmd.Short = `Get the restrict workspace admins setting.`
	cmd.Long = `Get the restrict workspace admins setting.
  
  Gets the restrict workspace admins setting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Settings.RestrictWorkspaceAdmins().Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*settings.UpdateRestrictWorkspaceAdminsSettingRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq settings.UpdateRestrictWorkspaceAdminsSettingRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update"
	cmd.Short = `Update the restrict workspace admins setting.`
	cmd.Long = `Update the restrict workspace admins setting.
  
  Updates the restrict workspace admins setting for the workspace. A fresh etag
  needs to be provided in PATCH requests (as part of the setting field). The
  etag can be retrieved by making a GET request before the PATCH request. If
  the setting is updated concurrently, PATCH fails with 409 and the request
  must be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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

		response, err := w.Settings.RestrictWorkspaceAdmins().Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service RestrictWorkspaceAdmins
