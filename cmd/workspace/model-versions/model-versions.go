// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package model_versions

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucregisteredmodels "github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	sdkucregisteredmodels "github.com/databricks/sdk-go/uc/registeredmodels/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*sdkucregisteredmodels.DeleteModelVersionRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkucregisteredmodels.DeleteModelVersionRequest

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

		deleteReq.FullNameArg = new(args[0])
		var deleteVersionArg int64
		_, err = fmt.Sscan(args[1], &deleteVersionArg)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}
		deleteReq.VersionArg = &deleteVersionArg

		sdkReq := &deleteReq
		response, err := w.RegisteredModels.DeleteModelVersion(ctx, *sdkReq)
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
	*ucregisteredmodels.GetModelVersionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq ucregisteredmodels.GetModelVersionRequest

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeAliases, "include-aliases", `Whether to include aliases associated with the model version in the response.`)
	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeBrowse, "include-browse", `Whether to include model versions in the response for which the principal can only access selective metadata for.`)

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

		getReq.FullNameArg = new(args[0])
		var getVersionArg int64
		_, err = fmt.Sscan(args[1], &getVersionArg)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}
		getReq.VersionArg = &getVersionArg

		sdkReq, err := ucregisteredmodels.GetModelVersionRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.GetModelVersion(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.ModelVersionInfoFromSDK(response)
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

// start get-by-alias command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getByAliasOverrides []func(
	*cobra.Command,
	*ucregisteredmodels.GetModelVersionByAliasRequest,
)

func newGetByAlias() *cobra.Command {
	cmd := &cobra.Command{}
	var getByAliasReq ucregisteredmodels.GetModelVersionByAliasRequest

	flags.BoolPtrVar(cmd.Flags(), &getByAliasReq.IncludeAliases, "include-aliases", `Whether to include aliases associated with the model version in the response.`)

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

		getByAliasReq.FullNameArg = new(args[0])
		getByAliasReq.AliasArg = new(args[1])

		sdkReq, err := ucregisteredmodels.GetModelVersionByAliasRequestToSDK(&getByAliasReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.GetModelVersionByAlias(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.ModelVersionInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucregisteredmodels.ListModelVersionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucregisteredmodels.ListModelVersionsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeBrowse, "include-browse", `Whether to include model versions in the response for which the principal can only access selective metadata for.`)
	flags.Int64PtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of model versions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list FULL_NAME"
	cmd.Short = `List Model Versions.`
	cmd.Long = `List Model Versions.

  List the model versions of the specified registered model, identified by its
  full three-level name (catalog.schema.model).

  The returned models are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the model versions. A
  regular user needs to be the owner or have the **EXECUTE** privilege on the
  parent registered model to receive the model versions in the response. For the
  latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.

  There is no guarantee of a specific ordering of the elements in the response.
  The elements in the response will not contain any aliases or tags.

  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.

  Arguments:
    FULL_NAME: The full three-level name of the registered model under which to list
      model versions`

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

		listReq.FullNameArg = new(args[0])

		sdkReq, err := ucregisteredmodels.ListModelVersionsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.RegisteredModels.ListModelVersionsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucregisteredmodels.ModelVersionInfoFromSDK))
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
	*ucregisteredmodels.UpdateModelVersionRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucregisteredmodels.UpdateModelVersionRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: aliases
	flags.StringPtrVar(cmd.Flags(), &updateReq.CatalogName, "catalog-name", `The name of the catalog containing the model version.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `The comment attached to the model version.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.CreatedAt, "created-at", ``)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CreatedBy, "created-by", `The identifier of the user who created the model version.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Id, "id", `The unique identifier of the model version.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.MetastoreId, "metastore-id", `The unique identifier of the metastore containing the model version.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.ModelName, "model-name", `The name of the parent registered model of the model version, relative to parent schema.`)
	// TODO: complex arg: model_version_dependencies
	flags.StringPtrVar(cmd.Flags(), &updateReq.RunId, "run-id", `MLflow run ID used when creating the model version, if source was generated by an experiment run stored in an MLflow tracking server.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.RunWorkspaceId, "run-workspace-id", `ID of the Databricks workspace containing the MLflow run that generated this model version, if applicable.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.SchemaName, "schema-name", `The name of the schema containing the model version, relative to parent catalog.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Source, "source", `URI indicating the location of the source artifacts (files) for the model version.`)
	flags.EnumVar(cmd.Flags(), &updateReq.Status, "status", `Current status of the model version. Supported values: [FAILED_REGISTRATION, MODEL_VERSION_STATUS_UNKNOWN, PENDING_REGISTRATION, READY]`, "ModelVersionInfoStatus", "FAILED_REGISTRATION", "MODEL_VERSION_STATUS_UNKNOWN", "PENDING_REGISTRATION", "READY")
	flags.StringPtrVar(cmd.Flags(), &updateReq.StorageLocation, "storage-location", `The storage location on the cloud under which model version data files are stored.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.UpdatedAt, "updated-at", ``)
	flags.StringPtrVar(cmd.Flags(), &updateReq.UpdatedBy, "updated-by", `The identifier of the user who updated the model version last time.`)

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
		updateReq.FullNameArg = new(args[0])
		var updateVersionArg int64
		_, err = fmt.Sscan(args[1], &updateVersionArg)
		if err != nil {
			return fmt.Errorf("invalid VERSION: %s", args[1])
		}
		updateReq.VersionArg = &updateVersionArg

		sdkReq, err := ucregisteredmodels.UpdateModelVersionRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.UpdateModelVersion(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.ModelVersionInfoFromSDK(response)
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

// end service ModelVersions
