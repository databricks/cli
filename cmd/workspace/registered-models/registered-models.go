// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package registered_models

import (
	"errors"
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
		Use:   "registered-models",
		Short: `Databricks provides a hosted version of MLflow Model Registry in Unity Catalog.`,
		Long: `Databricks provides a hosted version of MLflow Model Registry in Unity
  Catalog. Models in Unity Catalog provide centralized access control, auditing,
  lineage, and discovery of ML models across Databricks workspaces.

  An MLflow registered model resides in the third layer of Unity Catalog’s
  three-level namespace. Registered models contain model versions, which
  correspond to actual ML models (MLflow models). Creating new model versions
  requires use of the MLflow Python client. After model versions are created,
  you can load them for batch inference using MLflow Python client APIs, or
  deploy them for real-time serving using Databricks Model Serving.

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

  Note: The securable type for models is FUNCTION. When using REST APIs (for
  example, tagging, grants) that specify a securable type, use FUNCTION as the
  securable type.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*ucregisteredmodels.CreateRegisteredModelRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucregisteredmodels.CreateRegisteredModelRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: aliases
	flags.StringPtrVar(cmd.Flags(), &createReq.CatalogName, "catalog-name", `The name of the catalog where the schema and the registered model reside.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `The comment attached to the registered model.`)
	flags.Int64PtrVar(cmd.Flags(), &createReq.CreatedAt, "created-at", `Creation timestamp of the registered model in milliseconds since the Unix epoch.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.CreatedBy, "created-by", `The identifier of the user who created the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.FullName, "full-name", `The three-level (fully qualified) name of the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.MetastoreId, "metastore-id", `The unique identifier of the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Name, "name", `The name of the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Owner, "owner", `The identifier of the user who owns the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.SchemaName, "schema-name", `The name of the schema where the registered model resides.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageLocation, "storage-location", `The storage location on the cloud under which model version data files are stored.`)
	flags.Int64PtrVar(cmd.Flags(), &createReq.UpdatedAt, "updated-at", `Last-update timestamp of the registered model in milliseconds since the Unix epoch.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.UpdatedBy, "updated-by", `The identifier of the user who updated the registered model last time.`)

	cmd.Use = "create"
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
  schema.`

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

		sdkReq, err := ucregisteredmodels.CreateRegisteredModelRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.CreateRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.RegisteredModelInfoFromSDK(response)
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
	*sdkucregisteredmodels.DeleteRegisteredModelRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkucregisteredmodels.DeleteRegisteredModelRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No FULL_NAME argument specified. Loading names for Registered Models drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.RegisteredModels.ListRegisteredModelsIter(ctx, sdkucregisteredmodels.ListRegisteredModelsRequest{}), []string{"Name"}, []string{"FullName"})
			sp.Close()
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
			return errors.New("expected to have the three-level (fully qualified) name of the registered model")
		}
		deleteReq.FullNameArg = new(args[0])

		sdkReq := &deleteReq
		response, err := w.RegisteredModels.DeleteRegisteredModel(ctx, *sdkReq)
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

// start delete-alias command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAliasOverrides []func(
	*cobra.Command,
	*sdkucregisteredmodels.DeleteRegisteredModelAliasRequest,
)

func newDeleteAlias() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteAliasReq sdkucregisteredmodels.DeleteRegisteredModelAliasRequest

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

		deleteAliasReq.FullNameArg = new(args[0])
		deleteAliasReq.AliasArg = new(args[1])

		sdkReq := &deleteAliasReq
		response, err := w.RegisteredModels.DeleteRegisteredModelAlias(ctx, *sdkReq)
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
	*ucregisteredmodels.GetRegisteredModelRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq ucregisteredmodels.GetRegisteredModelRequest

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeAliases, "include-aliases", `Whether to include registered model aliases in the response.`)
	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeBrowse, "include-browse", `Whether to include registered models in the response for which the principal can only access selective metadata for.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No FULL_NAME argument specified. Loading names for Registered Models drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.RegisteredModels.ListRegisteredModelsIter(ctx, sdkucregisteredmodels.ListRegisteredModelsRequest{}), []string{"Name"}, []string{"FullName"})
			sp.Close()
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
			return errors.New("expected to have the three-level (fully qualified) name of the registered model")
		}
		getReq.FullNameArg = new(args[0])

		sdkReq, err := ucregisteredmodels.GetRegisteredModelRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.GetRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.RegisteredModelInfoFromSDK(response)
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
	*ucregisteredmodels.ListRegisteredModelsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucregisteredmodels.ListRegisteredModelsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.CatalogName, "catalog-name", `The identifier of the catalog under which to list registered models.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeBrowse, "include-browse", `Whether to include registered models in the response for which the principal can only access selective metadata for.`)
	flags.Int64PtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Max number of registered models to return.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.SchemaName, "schema-name", `The identifier of the schema under which to list registered models.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List Registered Models.`
	cmd.Long = `List Registered Models.

  List registered models. You can list registered models under a particular
  schema, or list all registered models in the current metastore.

  The returned models are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the registered models. A
  regular user needs to be the owner or have the **EXECUTE** privilege on the
  registered model to receive the registered models in the response. For the
  latter case, the caller must also be the owner or have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema.

  There is no guarantee of a specific ordering of the elements in the response.

  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.`

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

		sdkReq, err := ucregisteredmodels.ListRegisteredModelsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.RegisteredModels.ListRegisteredModelsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucregisteredmodels.RegisteredModelInfoFromSDK))
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
	*ucregisteredmodels.SetRegisteredModelAliasRequest,
)

func newSetAlias() *cobra.Command {
	cmd := &cobra.Command{}
	var setAliasReq ucregisteredmodels.SetRegisteredModelAliasRequest

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
    FULL_NAME: The three-level (fully qualified) name of the registered model
    ALIAS: The name of the alias
    VERSION_NUM: The version number of the model version to which the alias points`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only FULL_NAME, ALIAS as positional arguments. Provide 'version_num' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setAliasJson.Unmarshal(&setAliasReq)
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
		setAliasReq.FullNameArg = new(args[0])
		setAliasReq.AliasArg = new(args[1])
		if !cmd.Flags().Changed("json") {
			var setAliasVersionNumArg int64
			_, err = fmt.Sscan(args[2], &setAliasVersionNumArg)
			if err != nil {
				return fmt.Errorf("invalid VERSION_NUM: %s", args[2])
			}
			setAliasReq.VersionNum = &setAliasVersionNumArg
		}

		sdkReq, err := ucregisteredmodels.SetRegisteredModelAliasRequestToSDK(&setAliasReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.SetRegisteredModelAlias(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.RegisteredModelAliasInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucregisteredmodels.UpdateRegisteredModelRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucregisteredmodels.UpdateRegisteredModelRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: aliases
	flags.StringPtrVar(cmd.Flags(), &updateReq.CatalogName, "catalog-name", `The name of the catalog where the schema and the registered model reside.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `The comment attached to the registered model.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.CreatedAt, "created-at", `Creation timestamp of the registered model in milliseconds since the Unix epoch.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CreatedBy, "created-by", `The identifier of the user who created the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.MetastoreId, "metastore-id", `The unique identifier of the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Name, "name", `The name of the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `The identifier of the user who owns the registered model.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.SchemaName, "schema-name", `The name of the schema where the registered model resides.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.StorageLocation, "storage-location", `The storage location on the cloud under which model version data files are stored.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.UpdatedAt, "updated-at", `Last-update timestamp of the registered model in milliseconds since the Unix epoch.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.UpdatedBy, "updated-by", `The identifier of the user who updated the registered model last time.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No FULL_NAME argument specified. Loading names for Registered Models drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.RegisteredModels.ListRegisteredModelsIter(ctx, sdkucregisteredmodels.ListRegisteredModelsRequest{}), []string{"Name"}, []string{"FullName"})
			sp.Close()
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
			return errors.New("expected to have the three-level (fully qualified) name of the registered model")
		}
		updateReq.FullNameArg = new(args[0])

		sdkReq, err := ucregisteredmodels.UpdateRegisteredModelRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.RegisteredModels.UpdateRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucregisteredmodels.RegisteredModelInfoFromSDK(response)
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

// end service RegisteredModels
