// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package model_versions

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
		Use:   "model-versions",
		Short: `Databricks provides a hosted version of MLflow Model Registry in Unity Catalog.`,
		Long: `Databricks provides a hosted version of MLflow Model Registry in Unity
  Catalog. Models in Unity Catalog provide centralized access control, auditing,
  lineage, and discovery of ML models across Databricks workspaces.
  
  This API reference documents the REST endpoints for managing model versions in
  Unity Catalog. For more details, see the [registered models API
  docs](/api/workspace/registeredmodels).`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetByAlias())
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
	*catalog.DeleteModelVersionRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteModelVersionRequest

	cmd.Use = "delete FULL_NAME VERSION"
	cmd.Short = `Delete a Model Version.`
	cmd.Long = `Delete a Model Version.
  
  Deletes a model version from the specified registered model. Any aliases
  assigned to the model version will also be deleted.
  
  The caller must be a metastore admin or an owner of the parent registered
  model. For the latter case, the caller must also be the owner or have the
  **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA**
  privilege on the parent schema.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the model version
    VERSION: The integer version number of the model version`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.FullName = args[0]
		_, err = fmt.Sscan(args[1], &deleteReq.Version)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}

		err = w.ModelVersions.Delete(ctx, deleteReq)
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
	*catalog.GetModelVersionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetModelVersionRequest

	cmd.Flags().BoolVar(&getReq.IncludeAliases, "include-aliases", getReq.IncludeAliases, `Whether to include aliases associated with the model version in the response. Wire name: 'include_aliases'.`)
	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include model versions in the response for which the principal can only access selective metadata for. Wire name: 'include_browse'.`)

	cmd.Use = "get FULL_NAME VERSION"
	cmd.Short = `Get a Model Version.`
	cmd.Long = `Get a Model Version.
  
  Get a model version.
  
  The caller must be a metastore admin or an owner of (or have the **EXECUTE**
  privilege on) the parent registered model. For the latter case, the caller
  must also be the owner or have the **USE_CATALOG** privilege on the parent
  catalog and the **USE_SCHEMA** privilege on the parent schema.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the model version
    VERSION: The integer version number of the model version`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.FullName = args[0]
		_, err = fmt.Sscan(args[1], &getReq.Version)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}

		response, err := w.ModelVersions.Get(ctx, getReq)
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

// start get-by-alias command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getByAliasOverrides []func(
	*cobra.Command,
	*catalog.GetByAliasRequest,
)

func newGetByAlias() *cobra.Command {
	cmd := &cobra.Command{}

	var getByAliasReq catalog.GetByAliasRequest

	cmd.Flags().BoolVar(&getByAliasReq.IncludeAliases, "include-aliases", getByAliasReq.IncludeAliases, `Whether to include aliases associated with the model version in the response. Wire name: 'include_aliases'.`)

	cmd.Use = "get-by-alias FULL_NAME ALIAS"
	cmd.Short = `Get Model Version By Alias.`
	cmd.Long = `Get Model Version By Alias.
  
  Get a model version by alias.
  
  The caller must be a metastore admin or an owner of (or have the **EXECUTE**
  privilege on) the registered model. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema.

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

		getByAliasReq.FullName = args[0]
		getByAliasReq.Alias = args[1]

		response, err := w.ModelVersions.GetByAlias(ctx, getByAliasReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getByAliasOverrides {
		fn(cmd, &getByAliasReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*catalog.ListModelVersionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListModelVersionsRequest

	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include model versions in the response for which the principal can only access selective metadata for. Wire name: 'include_browse'.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of model versions to return. Wire name: 'max_results'.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query. Wire name: 'page_token'.`)

	cmd.Use = "list FULL_NAME"
	cmd.Short = `List Model Versions.`
	cmd.Long = `List Model Versions.
  
  List model versions. You can list model versions under a particular schema, or
  list all model versions in the current metastore.
  
  The returned models are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the model versions. A
  regular user needs to be the owner or have the **EXECUTE** privilege on the
  parent registered model to recieve the model versions in the response. For the
  latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.
  
  There is no guarantee of a specific ordering of the elements in the response.
  The elements in the response will not contain any aliases or tags.

  Arguments:
    FULL_NAME: The full three-level name of the registered model under which to list
      model versions`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.FullName = args[0]

		response := w.ModelVersions.List(ctx, listReq)
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
	*catalog.UpdateModelVersionRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateModelVersionRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: aliases
	cmd.Flags().StringVar(&updateReq.CatalogName, "catalog-name", updateReq.CatalogName, `The name of the catalog containing the model version. Wire name: 'catalog_name'.`)
	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `The comment attached to the model version. Wire name: 'comment'.`)
	cmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", updateReq.CreatedAt, `Wire name: 'created_at'.`)
	cmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", updateReq.CreatedBy, `The identifier of the user who created the model version. Wire name: 'created_by'.`)
	cmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `The unique identifier of the model version. Wire name: 'id'.`)
	cmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", updateReq.MetastoreId, `The unique identifier of the metastore containing the model version. Wire name: 'metastore_id'.`)
	cmd.Flags().StringVar(&updateReq.ModelName, "model-name", updateReq.ModelName, `The name of the parent registered model of the model version, relative to parent schema. Wire name: 'model_name'.`)
	// TODO: complex arg: model_version_dependencies
	cmd.Flags().StringVar(&updateReq.RunId, "run-id", updateReq.RunId, `MLflow run ID used when creating the model version, if source was generated by an experiment run stored in an MLflow tracking server. Wire name: 'run_id'.`)
	cmd.Flags().IntVar(&updateReq.RunWorkspaceId, "run-workspace-id", updateReq.RunWorkspaceId, `ID of the Databricks workspace containing the MLflow run that generated this model version, if applicable. Wire name: 'run_workspace_id'.`)
	cmd.Flags().StringVar(&updateReq.SchemaName, "schema-name", updateReq.SchemaName, `The name of the schema containing the model version, relative to parent catalog. Wire name: 'schema_name'.`)
	cmd.Flags().StringVar(&updateReq.Source, "source", updateReq.Source, `URI indicating the location of the source artifacts (files) for the model version. Wire name: 'source'.`)
	cmd.Flags().Var(&updateReq.Status, "status", `Current status of the model version. Supported values: [FAILED_REGISTRATION, MODEL_VERSION_STATUS_UNKNOWN, PENDING_REGISTRATION, READY]. Wire name: 'status'.`)
	cmd.Flags().StringVar(&updateReq.StorageLocation, "storage-location", updateReq.StorageLocation, `The storage location on the cloud under which model version data files are stored. Wire name: 'storage_location'.`)
	cmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", updateReq.UpdatedAt, `Wire name: 'updated_at'.`)
	cmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", updateReq.UpdatedBy, `The identifier of the user who updated the model version last time. Wire name: 'updated_by'.`)

	cmd.Use = "update FULL_NAME VERSION"
	cmd.Short = `Update a Model Version.`
	cmd.Long = `Update a Model Version.
  
  Updates the specified model version.
  
  The caller must be a metastore admin or an owner of the parent registered
  model. For the latter case, the caller must also be the owner or have the
  **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA**
  privilege on the parent schema.
  
  Currently only the comment of the model version can be updated.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the model version
    VERSION: The integer version number of the model version`

	cmd.Annotations = make(map[string]string)

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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.FullName = args[0]
		_, err = fmt.Sscan(args[1], &updateReq.Version)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}

		response, err := w.ModelVersions.Update(ctx, updateReq)
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

// end service ModelVersions
