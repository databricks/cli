// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package registered_models

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
		Use:   "registered-models",
		Short: `Databricks provides a hosted version of MLflow Model Registry in Unity Catalog.`,
		Long: `Databricks provides a hosted version of MLflow Model Registry in Unity
  Catalog. Models in Unity Catalog provide centralized access control, auditing,
  lineage, and discovery of ML models across Databricks workspaces.
  
  An MLflow registered model resides in the third layer of Unity Catalog’s
  three-level namespace. Registered models contain model versions, which
  correspond to actual ML models (MLflow models). Creating new model versions
  currently requires use of the MLflow Python client. Once model versions are
  created, you can load them for batch inference using MLflow Python client
  APIs, or deploy them for real-time serving using Databricks Model Serving.
  
  All operations on registered models and model versions require USE_CATALOG
  permissions on the enclosing catalog and USE_SCHEMA permissions on the
  enclosing schema. In addition, the following additional privileges are
  required for various operations:
  
  * To create a registered model, users must additionally have the CREATE_MODEL
  permission on the target schema. * To view registered model or model version
  metadata, model version data files, or invoke a model version, users must
  additionally have the EXECUTE permission on the registered model * To update
  registered model or model version tags, users must additionally have APPLY TAG
  permissions on the registered model * To update other registered model or
  model version metadata (comments, aliases) create a new model version, or
  update permissions on the registered model, users must be owners of the
  registered model.
  
  Note: The securable type for models is "FUNCTION". When using REST APIs (e.g.
  tagging, grants) that specify a securable type, use "FUNCTION" as the
  securable type.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteAlias())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSetAlias())
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
	*catalog.CreateRegisteredModelRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateRegisteredModelRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `The comment attached to the registered model.`)
	cmd.Flags().StringVar(&createReq.StorageLocation, "storage-location", createReq.StorageLocation, `The storage location on the cloud under which model version data files are stored.`)

	cmd.Use = "create CATALOG_NAME SCHEMA_NAME NAME"
	cmd.Short = `Create a Registered Model.`
	cmd.Long = `Create a Registered Model.
  
  Creates a new registered model in Unity Catalog.
  
  File storage for model versions in the registered model will be located in the
  default location which is specified by the parent schema, or the parent
  catalog, or the Metastore.
  
  For registered model creation to succeed, the user must satisfy the following
  conditions: - The caller must be a metastore admin, or be the owner of the
  parent catalog and schema, or have the **USE_CATALOG** privilege on the parent
  catalog and the **USE_SCHEMA** privilege on the parent schema. - The caller
  must have the **CREATE MODEL** or **CREATE FUNCTION** privilege on the parent
  schema.

  Arguments:
    CATALOG_NAME: The name of the catalog where the schema and the registered model reside
    SCHEMA_NAME: The name of the schema where the registered model resides
    NAME: The name of the registered model`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'catalog_name', 'schema_name', 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
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
			createReq.CatalogName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.SchemaName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[2]
		}

		response, err := w.RegisteredModels.Create(ctx, createReq)
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
	*catalog.DeleteRegisteredModelRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteRegisteredModelRequest

	cmd.Use = "delete FULL_NAME"
	cmd.Short = `Delete a Registered Model.`
	cmd.Long = `Delete a Registered Model.
  
  Deletes a registered model and all its model versions from the specified
  parent catalog and schema.
  
  The caller must be a metastore admin or an owner of the registered model. For
  the latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the registered model`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Registered Models drop-down."
			names, err := w.RegisteredModels.RegisteredModelInfoNameToFullNameMap(ctx, catalog.ListRegisteredModelsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Registered Models drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the registered model")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the registered model")
		}
		deleteReq.FullName = args[0]

		err = w.RegisteredModels.Delete(ctx, deleteReq)
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

// start delete-alias command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAliasOverrides []func(
	*cobra.Command,
	*catalog.DeleteAliasRequest,
)

func newDeleteAlias() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteAliasReq catalog.DeleteAliasRequest

	cmd.Use = "delete-alias FULL_NAME ALIAS"
	cmd.Short = `Delete a Registered Model Alias.`
	cmd.Long = `Delete a Registered Model Alias.
  
  Deletes a registered model alias.
  
  The caller must be a metastore admin or an owner of the registered model. For
  the latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the registered model
    ALIAS: The name of the alias`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteAliasReq.FullName = args[0]
		deleteAliasReq.Alias = args[1]

		err = w.RegisteredModels.DeleteAlias(ctx, deleteAliasReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteAliasOverrides {
		fn(cmd, &deleteAliasReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetRegisteredModelRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetRegisteredModelRequest

	cmd.Flags().BoolVar(&getReq.IncludeAliases, "include-aliases", getReq.IncludeAliases, `Whether to include registered model aliases in the response.`)
	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include registered models in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get FULL_NAME"
	cmd.Short = `Get a Registered Model.`
	cmd.Long = `Get a Registered Model.
  
  Get a registered model.
  
  The caller must be a metastore admin or an owner of (or have the **EXECUTE**
  privilege on) the registered model. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the registered model`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Registered Models drop-down."
			names, err := w.RegisteredModels.RegisteredModelInfoNameToFullNameMap(ctx, catalog.ListRegisteredModelsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Registered Models drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the registered model")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the registered model")
		}
		getReq.FullName = args[0]

		response, err := w.RegisteredModels.Get(ctx, getReq)
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
	*catalog.ListRegisteredModelsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListRegisteredModelsRequest

	cmd.Flags().StringVar(&listReq.CatalogName, "catalog-name", listReq.CatalogName, `The identifier of the catalog under which to list registered models.`)
	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include registered models in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Max number of registered models to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque token to send for the next page of results (pagination).`)
	cmd.Flags().StringVar(&listReq.SchemaName, "schema-name", listReq.SchemaName, `The identifier of the schema under which to list registered models.`)

	cmd.Use = "list"
	cmd.Short = `List Registered Models.`
	cmd.Long = `List Registered Models.
  
  List registered models. You can list registered models under a particular
  schema, or list all registered models in the current metastore.
  
  The returned models are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the registered models. A
  regular user needs to be the owner or have the **EXECUTE** privilege on the
  registered model to recieve the registered models in the response. For the
  latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.
  
  There is no guarantee of a specific ordering of the elements in the response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.RegisteredModels.List(ctx, listReq)
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

// start set-alias command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setAliasOverrides []func(
	*cobra.Command,
	*catalog.SetRegisteredModelAliasRequest,
)

func newSetAlias() *cobra.Command {
	cmd := &cobra.Command{}

	var setAliasReq catalog.SetRegisteredModelAliasRequest
	var setAliasJson flags.JsonFlag

	cmd.Flags().Var(&setAliasJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-alias FULL_NAME ALIAS VERSION_NUM"
	cmd.Short = `Set a Registered Model Alias.`
	cmd.Long = `Set a Registered Model Alias.
  
  Set an alias on the specified registered model.
  
  The caller must be a metastore admin or an owner of the registered model. For
  the latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.

  Arguments:
    FULL_NAME: Full name of the registered model
    ALIAS: The name of the alias
    VERSION_NUM: The version number of the model version to which the alias points`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only FULL_NAME, ALIAS as positional arguments. Provide 'version_num' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setAliasJson.Unmarshal(&setAliasReq)
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
		setAliasReq.FullName = args[0]
		setAliasReq.Alias = args[1]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &setAliasReq.VersionNum)
			if err != nil {
				return fmt.Errorf("invalid VERSION_NUM: %s", args[2])
			}
		}

		response, err := w.RegisteredModels.SetAlias(ctx, setAliasReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setAliasOverrides {
		fn(cmd, &setAliasReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateRegisteredModelRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateRegisteredModelRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `The comment attached to the registered model.`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the registered model.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The identifier of the user who owns the registered model.`)

	cmd.Use = "update FULL_NAME"
	cmd.Short = `Update a Registered Model.`
	cmd.Long = `Update a Registered Model.
  
  Updates the specified registered model.
  
  The caller must be a metastore admin or an owner of the registered model. For
  the latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.
  
  Currently only the name, the owner or the comment of the registered model can
  be updated.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the registered model`

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
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Registered Models drop-down."
			names, err := w.RegisteredModels.RegisteredModelInfoNameToFullNameMap(ctx, catalog.ListRegisteredModelsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Registered Models drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the registered model")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the registered model")
		}
		updateReq.FullName = args[0]

		response, err := w.RegisteredModels.Update(ctx, updateReq)
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

// end service RegisteredModels
