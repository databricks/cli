// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package grants

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
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
	*catalog.GetGrantRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetGrantRequest

	cmd.Flags().IntVar(&getReq.MaxResults, "max-results", getReq.MaxResults, `Specifies the maximum number of privileges to return (page length).`)
	cmd.Flags().StringVar(&getReq.PageToken, "page-token", getReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)
	cmd.Flags().StringVar(&getReq.Principal, "principal", getReq.Principal, `If provided, only the permissions for the specified principal (user or group) are returned.`)

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

		getReq.SecurableType = args[0]
		getReq.FullName = args[1]

		response, err := w.Grants.Get(ctx, getReq)
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

// start get-effective command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEffectiveOverrides []func(
	*cobra.Command,
	*catalog.GetEffectiveRequest,
)

func newGetEffective() *cobra.Command {
	cmd := &cobra.Command{}

	var getEffectiveReq catalog.GetEffectiveRequest

	cmd.Flags().IntVar(&getEffectiveReq.MaxResults, "max-results", getEffectiveReq.MaxResults, `Specifies the maximum number of privileges to return (page length).`)
	cmd.Flags().StringVar(&getEffectiveReq.PageToken, "page-token", getEffectiveReq.PageToken, `Opaque token for the next page of results (pagination).`)
	cmd.Flags().StringVar(&getEffectiveReq.Principal, "principal", getEffectiveReq.Principal, `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

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

		getEffectiveReq.SecurableType = args[0]
		getEffectiveReq.FullName = args[1]

		response, err := w.Grants.GetEffective(ctx, getEffectiveReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
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
	*catalog.ListPrivilegeAssignmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListPrivilegeAssignmentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Specifies the maximum number of privilege assignments to return (page length).`)
	cmd.Flags().StringVar(&listReq.Principal, "principal", listReq.Principal, `If provided, only the permissions for the specified principal (user or group) are returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list SECURABLE_TYPE FULL_NAME"
	cmd.Short = `*Public Preview* List permissions.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List permissions.

  Lists the privilege assignments for a securable. Does not include inherited
  privileges. Paginated version of Get Permissions API.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.SecurableType = args[0]
		listReq.FullName = args[1]

		response := w.Grants.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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

// start list-effective command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEffectiveOverrides []func(
	*cobra.Command,
	*catalog.ListEffectivePrivilegeAssignmentsRequest,
)

func newListEffective() *cobra.Command {
	cmd := &cobra.Command{}

	var listEffectiveReq catalog.ListEffectivePrivilegeAssignmentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEffectiveLimit int

	cmd.Flags().IntVar(&listEffectiveReq.PageSize, "page-size", listEffectiveReq.PageSize, `Specifies the maximum number of privilege assignments to return (page length).`)
	cmd.Flags().StringVar(&listEffectiveReq.Principal, "principal", listEffectiveReq.Principal, `If provided, only the effective permissions for the specified principal (user or group) are returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEffectiveLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listEffectiveReq.PageToken, "page-token", listEffectiveReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-effective SECURABLE_TYPE FULL_NAME"
	cmd.Short = `*Public Preview* List effective permissions.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List effective permissions.

  Lists the effective privilege assignments for a securable. Includes inherited
  privileges. Paginated version of Get Effective Permissions API.

  Arguments:
    SECURABLE_TYPE: Type of securable.
    FULL_NAME: Full name of securable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listEffectiveReq.SecurableType = args[0]
		listEffectiveReq.FullName = args[1]

		response := w.Grants.ListEffective(ctx, listEffectiveReq)
		if listEffectiveLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEffectiveLimit)
		}
		if listEffectiveLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEffectiveLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*catalog.UpdatePermissions,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdatePermissions
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	cmd.Flags().BoolVar(&updateReq.OmitPermissionsInResponse, "omit-permissions-in-response", updateReq.OmitPermissionsInResponse, `Optional, default false.`)

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
		updateReq.SecurableType = args[0]
		updateReq.FullName = args[1]

		response, err := w.Grants.Update(ctx, updateReq)
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

	// Register --generate-skeleton after overrides so it wraps any RunE they
	// installed; --generate-skeleton then short-circuits the whole command.
	root.RegisterGenerateSkeleton(cmd, &updateReq)

	return cmd
}

// end service Grants
