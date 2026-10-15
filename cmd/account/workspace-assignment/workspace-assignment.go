// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_assignment

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*sdkaccessmanagement.DeleteWorkspacePermissionAssignmentRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkaccessmanagement.DeleteWorkspacePermissionAssignmentRequest

	cmd.Use = "delete WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Delete permissions assignment.`
	cmd.Long = `Delete permissions assignment.

  Deletes the workspace permissions assignment in a given account and workspace
  for the specified principal.

  Arguments:
    WORKSPACE_ID: The workspace ID for the account.
    PRINCIPAL_ID: The ID of the user, service principal, or group.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		var deleteWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &deleteWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		deleteReq.WorkspaceId = &deleteWorkspaceIdArg
		var deletePrincipalIdArg int64
		_, err = fmt.Sscan(args[1], &deletePrincipalIdArg)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}
		deleteReq.PrincipalId = &deletePrincipalIdArg

		sdkReq := &deleteReq
		response, err := a.AccessManagement.DeleteWorkspacePermissionAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*sdkaccessmanagement.ListWorkspacePermissionsRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkaccessmanagement.ListWorkspacePermissionsRequest

	cmd.Use = "get WORKSPACE_ID"
	cmd.Short = `List workspace permissions.`
	cmd.Long = `List workspace permissions.

  Get an array of workspace permissions for the specified account and workspace.

  Arguments:
    WORKSPACE_ID: The workspace ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		var getWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &getWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		getReq.WorkspaceId = &getWorkspaceIdArg

		sdkReq := &getReq
		response, err := a.AccessManagement.ListWorkspacePermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListWorkspacePermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.ListWorkspacePermissionAssignmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq sdkaccessmanagement.ListWorkspacePermissionAssignmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list WORKSPACE_ID"
	cmd.Short = `Get permission assignments.`
	cmd.Long = `Get permission assignments.

  Get the permission assignments for the specified Databricks account and
  Databricks workspace.

  Arguments:
    WORKSPACE_ID: The workspace ID for the account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		var listWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &listWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		listReq.WorkspaceId = &listWorkspaceIdArg

		sdkReq := &listReq
		response, err := a.AccessManagement.ListWorkspacePermissionAssignments(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := accessmanagement.ListWorkspacePermissionAssignmentsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*accessmanagement.UpdateWorkspacePermissionAssignmentRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq accessmanagement.UpdateWorkspacePermissionAssignmentRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		var updateWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &updateWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		updateReq.WorkspaceId = &updateWorkspaceIdArg
		var updatePrincipalIdArg int64
		_, err = fmt.Sscan(args[1], &updatePrincipalIdArg)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}
		updateReq.PrincipalId = &updatePrincipalIdArg

		sdkReq, err := accessmanagement.UpdateWorkspacePermissionAssignmentRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.AccessManagement.UpdateWorkspacePermissionAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.WorkspacePermissionAssignmentOutputFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
