// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastore_assignments

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucmetastores "github.com/databricks/cli/libs/sdktypes/uc/metastores/v1"
	sdkucmetastores "github.com/databricks/sdk-go/uc/metastores/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metastore-assignments",
		Short:   `These APIs manage metastore assignments to a workspace.`,
		Long:    `These APIs manage metastore assignments to a workspace.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
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

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*ucmetastores.AccountsCreateMetastoreAssignmentRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucmetastores.AccountsCreateMetastoreAssignmentRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_assignment

	cmd.Use = "create WORKSPACE_ID METASTORE_ID"
	cmd.Short = `Assigns a workspace to a metastore.`
	cmd.Long = `Assigns a workspace to a metastore.

  Creates an assignment to a metastore for a workspace

  Arguments:
    WORKSPACE_ID: Workspace ID.
    METASTORE_ID: Unity Catalog metastore ID`

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
			diags := createJson.Unmarshal(&createReq)
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
		var createWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &createWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		createReq.WorkspaceId = &createWorkspaceIdArg
		createReq.MetastoreId = new(args[1])

		sdkReq, err := ucmetastores.AccountsCreateMetastoreAssignmentRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Metastores.CreateAccountsMetastoreAssignment(ctx, *sdkReq)
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
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkucmetastores.AccountsDeleteMetastoreAssignmentRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkucmetastores.AccountsDeleteMetastoreAssignmentRequest

	cmd.Use = "delete WORKSPACE_ID METASTORE_ID"
	cmd.Short = `Delete a metastore assignment.`
	cmd.Long = `Delete a metastore assignment.

  Deletes a metastore assignment to a workspace, leaving the workspace with no
  metastore.

  Arguments:
    WORKSPACE_ID: Workspace ID.
    METASTORE_ID: Unity Catalog metastore ID`

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
		deleteReq.MetastoreId = new(args[1])

		sdkReq := &deleteReq
		response, err := a.Metastores.DeleteAccountsMetastoreAssignment(ctx, *sdkReq)
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
	*sdkucmetastores.AccountsGetMetastoreAssignmentRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucmetastores.AccountsGetMetastoreAssignmentRequest

	cmd.Use = "get WORKSPACE_ID"
	cmd.Short = `Gets the metastore assignment for a workspace.`
	cmd.Long = `Gets the metastore assignment for a workspace.

  Gets the metastore assignment, if any, for the workspace specified by ID. If
  the workspace is assigned a metastore, the mapping will be returned. If no
  metastore is assigned to the workspace, the assignment will not be found and a
  404 returned.

  Arguments:
    WORKSPACE_ID: Workspace ID.`

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
		response, err := a.Metastores.GetMetastoreAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.AccountsGetMetastoreAssignmentResponseFromSDK(response)
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
	*sdkucmetastores.AccountsListWorkspaceIdsForMetastoreRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq sdkucmetastores.AccountsListWorkspaceIdsForMetastoreRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list METASTORE_ID"
	cmd.Short = `Get all workspaces assigned to a metastore.`
	cmd.Long = `Get all workspaces assigned to a metastore.

  Gets a list of all Databricks workspace IDs that have been assigned to given
  metastore.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

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

		listReq.MetastoreId = new(args[0])

		sdkReq := &listReq
		response, err := a.Metastores.ListMetastoreAssignments(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := ucmetastores.AccountsListWorkspaceIdsForMetastoreResponseFromSDK(response)
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
	*ucmetastores.AccountsUpdateMetastoreAssignmentRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucmetastores.AccountsUpdateMetastoreAssignmentRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: metastore_assignment

	cmd.Use = "update WORKSPACE_ID METASTORE_ID"
	cmd.Short = `Updates a metastore assignment to a workspace.`
	cmd.Long = `Updates a metastore assignment to a workspace.

  Updates an assignment to a metastore for a workspace. Currently, only the
  default catalog may be updated.

  Arguments:
    WORKSPACE_ID: Workspace ID.
    METASTORE_ID: Unity Catalog metastore ID`

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
		updateReq.MetastoreId = new(args[1])

		sdkReq, err := ucmetastores.AccountsUpdateMetastoreAssignmentRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.Metastores.UpdateAccountsMetastoreAssignment(ctx, *sdkReq)
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
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service AccountMetastoreAssignments
