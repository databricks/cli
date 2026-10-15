// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucgrants "github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grants",
		Short: `In Unity Catalog, data is secure by default.`,
		Long: `In Unity Catalog, data is secure by default. Initially, users have no access
  to data in a metastore. Access can be granted by either a metastore admin, the
  owner of an object, or the owner of the catalog or schema that contains the
  object. Securable objects in Unity Catalog are hierarchical and privileges are
  inherited downward.

  This means that granting a privilege on the catalog automatically grants the
  privilege to all current and future objects within the catalog. Similarly,
  privileges granted on a schema are inherited by all current and future objects
  within that schema.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetEffective())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListEffective())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*ucgrants.GetPermissionsRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq ucgrants.GetPermissionsRequest

	flags.IntPtrVar(cmd.Flags(), &getReq.MaxResults, "max-results", `Specifies the maximum number of privileges to return (page length).`)
	flags.StringPtrVar(cmd.Flags(), &getReq.PageToken, "page-token", `Opaque pagination token to go to next page based on previous query.`)
	flags.StringPtrVar(cmd.Flags(), &getReq.Principal, "principal", `If provided, only the permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get permissions.`
	cmd.Long = `Get permissions.

  Gets the permissions for a securable. Does not include inherited permissions.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.SecurableType = new(args[0])
		getReq.SecurableFullName = new(args[1])

		sdkReq, err := ucgrants.GetPermissionsRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.Grants.GetPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucgrants.GetPermissionsResponseFromSDK(response)
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

// start get-effective command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEffectiveOverrides []func(
	*cobra.Command,
	*ucgrants.GetEffectivePermissionsRequest,
)

func newGetEffective() *cobra.Command {
	cmd := &cobra.Command{}
	var getEffectiveReq ucgrants.GetEffectivePermissionsRequest

	flags.IntPtrVar(cmd.Flags(), &getEffectiveReq.MaxResults, "max-results", `Specifies the maximum number of privileges to return (page length).`)
	flags.StringPtrVar(cmd.Flags(), &getEffectiveReq.PageToken, "page-token", `Opaque token for the next page of results (pagination).`)
	flags.StringPtrVar(cmd.Flags(), &getEffectiveReq.Principal, "principal", `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

	cmd.Use = "get-effective SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Get effective permissions.`
	cmd.Long = `Get effective permissions.

  Gets the effective permissions for a securable. Includes inherited permissions
  from any parent securables.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getEffectiveReq.SecurableType = new(args[0])
		getEffectiveReq.SecurableFullName = new(args[1])

		sdkReq, err := ucgrants.GetEffectivePermissionsRequestToSDK(&getEffectiveReq)
		if err != nil {
			return err
		}
		response, err := w.Grants.GetEffectivePermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucgrants.GetEffectivePermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEffectiveOverrides {
		fn(cmd, &getEffectiveReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*ucgrants.ListPrivilegeAssignmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucgrants.ListPrivilegeAssignmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Specifies the maximum number of privilege assignments to return (page length).`)
	flags.StringPtrVar(cmd.Flags(), &listReq.Principal, "principal", `If provided, only the permissions for the specified principal (user or group) are returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list SECURABLE_TYPE FULL_NAME"
	cmd.Short = `List permissions.`
	cmd.Long = `List permissions.

  Lists the privilege assignments for a securable. Does not include inherited
  privileges. Paginated version of Get Permissions API.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listReq.SecurableType = new(args[0])
		listReq.FullName = new(args[1])

		sdkReq, err := ucgrants.ListPrivilegeAssignmentsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Grants.ListPrivilegeAssignmentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucgrants.PrivilegeAssignmentFromSDK))
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

// start list-effective command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEffectiveOverrides []func(
	*cobra.Command,
	*ucgrants.ListEffectivePrivilegeAssignmentsRequest,
)

func newListEffective() *cobra.Command {
	cmd := &cobra.Command{}
	var listEffectiveReq ucgrants.ListEffectivePrivilegeAssignmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEffectiveLimit int

	flags.IntPtrVar(cmd.Flags(), &listEffectiveReq.PageSize, "page-size", `Specifies the maximum number of privilege assignments to return (page length).`)
	flags.StringPtrVar(cmd.Flags(), &listEffectiveReq.Principal, "principal", `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEffectiveLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listEffectiveReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-effective SECURABLE_TYPE FULL_NAME"
	cmd.Short = `List effective permissions.`
	cmd.Long = `List effective permissions.

  Lists the effective privilege assignments for a securable. Includes inherited
  privileges. Paginated version of Get Effective Permissions API.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listEffectiveReq.SecurableType = new(args[0])
		listEffectiveReq.FullName = new(args[1])

		sdkReq, err := ucgrants.ListEffectivePrivilegeAssignmentsRequestToSDK(&listEffectiveReq)
		if err != nil {
			return err
		}
		if listEffectiveLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEffectiveLimit)
		}
		if listEffectiveLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEffectiveLimit)
		}
		response := w.Grants.ListEffectivePrivilegeAssignmentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucgrants.EffectivePrivilegeAssignmentFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEffectiveOverrides {
		fn(cmd, &listEffectiveReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*ucgrants.UpdatePermissionsRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucgrants.UpdatePermissionsRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	flags.BoolPtrVar(cmd.Flags(), &updateReq.OmitPermissionsInResponse, "omit-permissions-in-response", `Optional, default false.`)

	cmd.Use = "update SECURABLE_TYPE FULL_NAME"
	cmd.Short = `Update permissions.`
	cmd.Long = `Update permissions.

  Updates the permissions for a securable.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		updateReq.SecurableType = new(args[0])
		updateReq.SecurableFullName = new(args[1])

		sdkReq, err := ucgrants.UpdatePermissionsRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Grants.UpdatePermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucgrants.UpdatePermissionsResponseFromSDK(response)
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

// end service Grants
