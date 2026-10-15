// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_bindings

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucworkspacebindings "github.com/databricks/cli/libs/sdktypes/uc/workspacebindings/v1"
	sdkucworkspacebindings "github.com/databricks/sdk-go/uc/workspacebindings/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-bindings",
		Short: `A securable in Databricks can be configured as __OPEN__ or __ISOLATED__.`,
		Long: `A securable in Databricks can be configured as __OPEN__ or __ISOLATED__. An
  __OPEN__ securable can be accessed from any workspace, while an __ISOLATED__
  securable can only be accessed from a configured list of workspaces. This API
  allows you to configure (bind) securables to workspaces.

  NOTE: The __isolation_mode__ is configured for the securable itself (using its
  Update method) and the workspace bindings are only consulted when the
  securable's __isolation_mode__ is set to __ISOLATED__.

  A securable's workspace bindings can be configured by a metastore admin or the
  owner of the securable.

  The original path (/api/2.1/unity-catalog/workspace-bindings/catalogs/{name})
  is deprecated. Use the new path
  (/api/2.1/unity-catalog/bindings/{securable_type}/{securable_name}), which
  introduces the ability to bind a securable in READ_ONLY mode (catalogs only).

  Securable types that support binding: - catalog - storage_credential -
  credential - external_location`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetBindings())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdateBindings())

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
	*sdkucworkspacebindings.GetCatalogWorkspaceBindingsRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucworkspacebindings.GetCatalogWorkspaceBindingsRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get catalog workspace bindings.`
	cmd.Long = `Get catalog workspace bindings.

  Gets workspace bindings of the catalog. The caller must be a metastore admin
  or an owner of the catalog.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.CatalogName = new(args[0])

		sdkReq := &getReq
		response, err := w.WorkspaceBindings.GetCatalogWorkspaceBindings(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucworkspacebindings.GetCatalogWorkspaceBindingsResponseFromSDK(response)
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

// start get-bindings command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBindingsOverrides []func(
	*cobra.Command,
	*ucworkspacebindings.GetWorkspaceBindingsRequest,
)

func newGetBindings() *cobra.Command {
	cmd := &cobra.Command{}
	var getBindingsReq ucworkspacebindings.GetWorkspaceBindingsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var getBindingsLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&getBindingsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &getBindingsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &getBindingsReq.MaxResults, "max-results", `Maximum number of workspace bindings to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "get-bindings SECURABLE_TYPE SECURABLE_NAME"
	cmd.Short = `Get securable workspace bindings.`
	cmd.Long = `Get securable workspace bindings.

  Gets workspace bindings of the securable. The caller must be a metastore admin
  or an owner of the securable.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.

  Arguments:
    SECURABLE_TYPE: The type of the securable to bind to a workspace (catalog,
      storage_credential, credential, or external_location).
    SECURABLE_NAME: The name of the securable.`

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

		getBindingsReq.SecurableType = new(args[0])
		getBindingsReq.SecurableFullName = new(args[1])

		sdkReq, err := ucworkspacebindings.GetWorkspaceBindingsRequestToSDK(&getBindingsReq)
		if err != nil {
			return err
		}
		if getBindingsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", getBindingsLimit)
		}
		if getBindingsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, getBindingsLimit)
		}
		response := w.WorkspaceBindings.GetWorkspaceBindingsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucworkspacebindings.WorkspaceBindingInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBindingsOverrides {
		fn(cmd, &getBindingsReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*ucworkspacebindings.UpdateCatalogWorkspaceBindingsRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucworkspacebindings.UpdateCatalogWorkspaceBindingsRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: assign_workspaces
	// TODO: array: unassign_workspaces

	cmd.Use = "update NAME"
	cmd.Short = `Update catalog workspace bindings.`
	cmd.Long = `Update catalog workspace bindings.

  Updates workspace bindings of the catalog. The caller must be a metastore
  admin or an owner of the catalog.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
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
		updateReq.CatalogName = new(args[0])

		sdkReq, err := ucworkspacebindings.UpdateCatalogWorkspaceBindingsRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.WorkspaceBindings.UpdateCatalogWorkspaceBindings(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucworkspacebindings.UpdateCatalogWorkspaceBindingsResponseFromSDK(response)
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

// start update-bindings command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateBindingsOverrides []func(
	*cobra.Command,
	*ucworkspacebindings.UpdateWorkspaceBindingsRequest,
)

func newUpdateBindings() *cobra.Command {
	cmd := &cobra.Command{}
	var updateBindingsReq ucworkspacebindings.UpdateWorkspaceBindingsRequest

	var updateBindingsJson flags.JsonFlag

	cmd.Flags().Var(&updateBindingsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: add
	// TODO: array: remove

	cmd.Use = "update-bindings SECURABLE_TYPE SECURABLE_NAME"
	cmd.Short = `Update securable workspace bindings.`
	cmd.Long = `Update securable workspace bindings.

  Updates workspace bindings of the securable. The caller must be a metastore
  admin or an owner of the securable.

  Arguments:
    SECURABLE_TYPE: The type of the securable to bind to a workspace (catalog,
      storage_credential, credential, or external_location).
    SECURABLE_NAME: The name of the securable.`

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
			diags := updateBindingsJson.Unmarshal(&updateBindingsReq)
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
		updateBindingsReq.SecurableType = new(args[0])
		updateBindingsReq.SecurableFullName = new(args[1])

		sdkReq, err := ucworkspacebindings.UpdateWorkspaceBindingsRequestToSDK(&updateBindingsReq)
		if err != nil {
			return err
		}
		response, err := w.WorkspaceBindings.UpdateWorkspaceBindings(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucworkspacebindings.UpdateWorkspaceBindingsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateBindingsOverrides {
		fn(cmd, &updateBindingsReq)
	}

	return cmd
}

// end service WorkspaceBindings
