// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_assignment

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-assignment",
		Short: `The Workspace Permission Assignment API allows you to manage workspace permissions for principals in your account.`,
		Long: `The Workspace Permission Assignment API allows you to manage workspace
  permissions for principals in your account.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
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
	*iam.DeleteWorkspaceAssignmentRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq iam.DeleteWorkspaceAssignmentRequest

	cmd.Use = "delete WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Delete permissions assignment.`
	cmd.Long = `Delete permissions assignment.
  
  Deletes the workspace permissions assignment in a given account and workspace
  for the specified principal.

  Arguments:
    WORKSPACE_ID: The workspace ID for the account.
    PRINCIPAL_ID: The ID of the user, service principal, or group.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		err = a.WorkspaceAssignment.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
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
	*iam.GetWorkspaceAssignmentRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq iam.GetWorkspaceAssignmentRequest

	cmd.Use = "get WORKSPACE_ID"
	cmd.Short = `List workspace permissions.`
	cmd.Long = `List workspace permissions.
  
  Get an array of workspace permissions for the specified account and workspace.

  Arguments:
    WORKSPACE_ID: The workspace ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.WorkspaceAssignment.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*iam.ListWorkspaceAssignmentRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq iam.ListWorkspaceAssignmentRequest

	cmd.Use = "list WORKSPACE_ID"
	cmd.Short = `Get permission assignments.`
	cmd.Long = `Get permission assignments.
  
  Get the permission assignments for the specified Databricks account and
  Databricks workspace.

  Arguments:
    WORKSPACE_ID: The workspace ID for the account.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &listReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response := a.WorkspaceAssignment.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*iam.UpdateWorkspaceAssignments,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq iam.UpdateWorkspaceAssignments
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: permissions

	cmd.Use = "update WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Create or update permissions assignment.`
	cmd.Long = `Create or update permissions assignment.
  
  Creates or updates the workspace permissions assignment in a given account and
  workspace for the specified principal.

  Arguments:
    WORKSPACE_ID: The workspace ID.
    PRINCIPAL_ID: The ID of the user, service principal, or group.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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
		}
		_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &updateReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		response, err := a.WorkspaceAssignment.Update(ctx, updateReq)
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

// end service WorkspaceAssignment
