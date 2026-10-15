// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package catalogs

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	uccatalogs "github.com/databricks/cli/libs/sdktypes/uc/catalogs/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalogs",
		Short: `A catalog is the first layer of Unity Catalog’s three-level namespace.`,
		Long: `A catalog is the first layer of Unity Catalog’s three-level namespace.
  It’s used to organize your data assets. Users can see all catalogs on which
  they have been assigned the USE_CATALOG data permission.

  In Unity Catalog, admins and data stewards manage users and their access to
  data centrally across all of the workspaces in a Databricks account. Users in
  different workspaces can share access to the same data, depending on
  privileges granted centrally in Unity Catalog.`,
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
	*uccatalogs.CreateCatalogRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq uccatalogs.CreateCatalogRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.ConnectionName, "connection-name", `The name of the connection to an external data source.`)
	flags.Int64PtrVar(cmd.Flags(), &createReq.CustomMaxRetentionHours, "custom-max-retention-hours", `Custom maximum retention period in hours for the catalog.`)
	// TODO: complex arg: managed_encryption_settings
	// TODO: map via StringToStringVar: options
	// TODO: map via StringToStringVar: properties
	flags.StringPtrVar(cmd.Flags(), &createReq.ProviderName, "provider-name", `The name of delta sharing provider.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.ShareName, "share-name", `The name of the share under the share provider.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageRoot, "storage-root", `Storage root URL for managed tables within catalog.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a catalog.`
	cmd.Long = `Create a catalog.

  Creates a new catalog instance in the parent metastore if the caller is a
  metastore admin or has the **CREATE_CATALOG** privilege.

  Arguments:
    NAME: Name of catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		if !cmd.Flags().Changed("json") {
			createReq.Name = new(args[0])
		}

		sdkReq, err := uccatalogs.CreateCatalogRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Catalogs.CreateCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccatalogs.CatalogInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*uccatalogs.DeleteCatalogRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq uccatalogs.DeleteCatalogRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force deletion even if the catalog is not empty.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a catalog.`
	cmd.Long = `Delete a catalog.

  Deletes the catalog that matches the supplied name. The caller must be a
  metastore admin or the owner of the catalog.

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

		deleteReq.NameArg = new(args[0])

		sdkReq, err := uccatalogs.DeleteCatalogRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.Catalogs.DeleteCatalog(ctx, *sdkReq)
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
	*uccatalogs.GetCatalogRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq uccatalogs.GetCatalogRequest

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeBrowse, "include-browse", `Whether to include catalogs in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a catalog.`
	cmd.Long = `Get a catalog.

  Gets the specified catalog in a metastore. The caller must be a metastore
  admin, the owner of the catalog, or a user that has the **USE_CATALOG**
  privilege set for their account.

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

		getReq.NameArg = new(args[0])

		sdkReq, err := uccatalogs.GetCatalogRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.Catalogs.GetCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccatalogs.CatalogInfoFromSDK(response)
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
	*uccatalogs.ListCatalogsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq uccatalogs.ListCatalogsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeBrowse, "include-browse", `Whether to include catalogs in the response for which the principal can only access selective metadata for.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeUnbound, "include-unbound", `Whether to include catalogs not bound to the workspace.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of catalogs to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List catalogs.`
	cmd.Long = `List catalogs.

  Gets an array of catalogs in the metastore. If the caller is the metastore
  admin, all catalogs will be retrieved. Otherwise, only catalogs owned by the
  caller (or for which the caller has the **USE_CATALOG** privilege) will be
  retrieved. There is no guarantee of a specific ordering of the elements in the
  array.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := uccatalogs.ListCatalogsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Catalogs.ListCatalogsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, uccatalogs.CatalogInfoFromSDK))
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
	*uccatalogs.UpdateCatalogRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq uccatalogs.UpdateCatalogRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `User-provided free-form text description.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.CustomMaxRetentionHours, "custom-max-retention-hours", `Custom maximum retention period in hours for the catalog.`)
	flags.EnumPtrVar(cmd.Flags(), &updateReq.EnablePredictiveOptimization, "enable-predictive-optimization", `Whether predictive optimization should be enabled for this object and objects under it. Supported values: [DISABLE, ENABLE, INHERIT]`, "EnablePredictiveOptimization", "DISABLE", "ENABLE", "INHERIT")
	flags.EnumVar(cmd.Flags(), &updateReq.IsolationMode, "isolation-mode", `Whether the current securable is accessible from all workspaces or a specific set of workspaces. Supported values: [ISOLATED, OPEN]`, "CatalogIsolationMode", "ISOLATED", "OPEN")
	// TODO: complex arg: managed_encryption_settings
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the catalog.`)
	// TODO: map via StringToStringVar: options
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `Username of current owner of catalog.`)
	// TODO: map via StringToStringVar: properties

	cmd.Use = "update NAME"
	cmd.Short = `Update a catalog.`
	cmd.Long = `Update a catalog.

  Updates the catalog that matches the supplied name. The caller must be either
  the owner of the catalog, or a metastore admin (when changing the owner field
  of the catalog).

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
		updateReq.NameArg = new(args[0])

		sdkReq, err := uccatalogs.UpdateCatalogRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Catalogs.UpdateCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccatalogs.CatalogInfoFromSDK(response)
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

// end service Catalogs
