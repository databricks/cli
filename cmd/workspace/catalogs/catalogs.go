// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package catalogs

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
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*catalog.CreateCatalog,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateCatalog
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&createReq.ConnectionName, "connection-name", createReq.ConnectionName, `The name of the connection to an external data source.`)
	// TODO: map via StringToStringVar: options
	// TODO: map via StringToStringVar: properties
	cmd.Flags().StringVar(&createReq.ProviderName, "provider-name", createReq.ProviderName, `The name of delta sharing provider.`)
	cmd.Flags().StringVar(&createReq.ShareName, "share-name", createReq.ShareName, `The name of the share under the share provider.`)
	cmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", createReq.StorageRoot, `Storage root URL for managed tables within catalog.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a catalog.`
	cmd.Long = `Create a catalog.
  
  Creates a new catalog instance in the parent metastore if the caller is a
  metastore admin or has the **CREATE_CATALOG** privilege.

  Arguments:
    NAME: Name of catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}

		response, err := w.Catalogs.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*catalog.DeleteCatalogRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteCatalogRequest

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the catalog is not empty.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a catalog.`
	cmd.Long = `Delete a catalog.
  
  Deletes the catalog that matches the supplied name. The caller must be a
  metastore admin or the owner of the catalog.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.Catalogs.Delete(ctx, deleteReq)
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
	*catalog.GetCatalogRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetCatalogRequest

	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include catalogs in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a catalog.`
	cmd.Long = `Get a catalog.
  
  Gets the specified catalog in a metastore. The caller must be a metastore
  admin, the owner of the catalog, or a user that has the **USE_CATALOG**
  privilege set for their account.

  Arguments:
    NAME: The name of the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.Catalogs.Get(ctx, getReq)
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
	*catalog.ListCatalogsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListCatalogsRequest

	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include catalogs in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of catalogs to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List catalogs.`
	cmd.Long = `List catalogs.
  
  Gets an array of catalogs in the metastore. If the caller is the metastore
  admin, all catalogs will be retrieved. Otherwise, only catalogs owned by the
  caller (or for which the caller has the **USE_CATALOG** privilege) will be
  retrieved. There is no guarantee of a specific ordering of the elements in the
  array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Catalogs.List(ctx, listReq)
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
	*catalog.UpdateCatalog,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateCatalog
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().Var(&updateReq.EnablePredictiveOptimization, "enable-predictive-optimization", `Whether predictive optimization should be enabled for this object and objects under it. Supported values: [DISABLE, ENABLE, INHERIT]`)
	cmd.Flags().Var(&updateReq.IsolationMode, "isolation-mode", `Whether the current securable is accessible from all workspaces or a specific set of workspaces. Supported values: [ISOLATED, OPEN]`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the catalog.`)
	// TODO: map via StringToStringVar: options
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of catalog.`)
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.Name = args[0]

		response, err := w.Catalogs.Update(ctx, updateReq)
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

// end service Catalogs
