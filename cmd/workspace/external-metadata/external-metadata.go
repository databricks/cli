// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_metadata

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucexternalmetadata "github.com/databricks/cli/libs/sdktypes/uc/externalmetadata/v1"
	sdkucexternalmetadata "github.com/databricks/sdk-go/uc/externalmetadata/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external-metadata",
		Short: `*Public Preview* External Metadata objects enable customers to register and manage metadata about external systems within Unity Catalog.`,
		Long: `This command is in Public Preview and may change without notice.

External Metadata objects enable customers to register and manage metadata
  about external systems within Unity Catalog.

  These APIs provide a standardized way to create, update, retrieve, list, and
  delete external metadata objects. Fine-grained authorization ensures that only
  users with appropriate permissions can view and manage external metadata
  objects.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreateExternalMetadata())
	cmd.AddCommand(newDeleteExternalMetadata())
	cmd.AddCommand(newGetExternalMetadata())
	cmd.AddCommand(newListExternalMetadata())
	cmd.AddCommand(newUpdateExternalMetadata())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-external-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createExternalMetadataOverrides []func(
	*cobra.Command,
	*ucexternalmetadata.CreateExternalMetadataRequest,
)

func newCreateExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var createExternalMetadataReq ucexternalmetadata.CreateExternalMetadataRequest
	createExternalMetadataReq.ExternalMetadata = &ucexternalmetadata.ExternalMetadata{}

	var createExternalMetadataJson flags.JsonFlag

	cmd.Flags().Var(&createExternalMetadataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	flags.StringPtrVar(cmd.Flags(), &createExternalMetadataReq.ExternalMetadata.Description, "description", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &createExternalMetadataReq.ExternalMetadata.Owner, "owner", `Owner of the external metadata object.`)
	// TODO: map via StringToStringVar: properties
	flags.StringPtrVar(cmd.Flags(), &createExternalMetadataReq.ExternalMetadata.Url, "url", `URL associated with the external metadata object.`)

	cmd.Use = "create-external-metadata NAME SYSTEM_TYPE ENTITY_TYPE"
	cmd.Short = `*Public Preview* Create an external metadata object.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create an external metadata object.

  Creates a new external metadata object in the parent metastore if the caller
  is a metastore admin or has the **CREATE_EXTERNAL_METADATA** privilege. Grants
  **BROWSE** to all account users upon creation by default.

  Arguments:
    NAME: Name of the external metadata object.
    SYSTEM_TYPE: Type of external system.
      Supported values: [
        AMAZON_REDSHIFT,
        AZURE_SYNAPSE,
        CONFLUENT,
        DATABRICKS,
        GOOGLE_BIGQUERY,
        KAFKA,
        LOOKER,
        MICROSOFT_FABRIC,
        MICROSOFT_SQL_SERVER,
        MONGODB,
        MYSQL,
        ORACLE,
        OTHER,
        POSTGRESQL,
        POWER_BI,
        SALESFORCE,
        SAP,
        SERVICENOW,
        SNOWFLAKE,
        STREAM_NATIVE,
        TABLEAU,
        TERADATA,
        WORKDAY,
      ]
    ENTITY_TYPE: Type of entity within the external system.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'system_type', 'entity_type' in your JSON input")
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
			diags := createExternalMetadataJson.Unmarshal(&createExternalMetadataReq.ExternalMetadata)
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
			createExternalMetadataReq.ExternalMetadata.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createExternalMetadataReq.ExternalMetadata.SystemType = ucexternalmetadata.SystemType(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createExternalMetadataReq.ExternalMetadata.EntityType = new(args[2])
		}

		sdkReq, err := ucexternalmetadata.CreateExternalMetadataRequestToSDK(&createExternalMetadataReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalMetadata.CreateExternalMetadataV2(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternalmetadata.ExternalMetadataFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createExternalMetadataOverrides {
		fn(cmd, &createExternalMetadataReq)
	}

	return cmd
}

// start delete-external-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExternalMetadataOverrides []func(
	*cobra.Command,
	*sdkucexternalmetadata.DeleteExternalMetadataRequest,
)

func newDeleteExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteExternalMetadataReq sdkucexternalmetadata.DeleteExternalMetadataRequest

	cmd.Use = "delete-external-metadata NAME"
	cmd.Short = `*Public Preview* Delete an external metadata object.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete an external metadata object.

  Deletes the external metadata object that matches the supplied name. The
  caller must be a metastore admin, the owner of the external metadata object,
  or a user that has the **MANAGE** privilege.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteExternalMetadataReq.Name = new(args[0])

		sdkReq := &deleteExternalMetadataReq
		err = w.ExternalMetadata.DeleteExternalMetadataV2(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteExternalMetadataOverrides {
		fn(cmd, &deleteExternalMetadataReq)
	}

	return cmd
}

// start get-external-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalMetadataOverrides []func(
	*cobra.Command,
	*sdkucexternalmetadata.GetExternalMetadataRequest,
)

func newGetExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalMetadataReq sdkucexternalmetadata.GetExternalMetadataRequest

	cmd.Use = "get-external-metadata NAME"
	cmd.Short = `*Public Preview* Get an external metadata object.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get an external metadata object.

  Gets the specified external metadata object in a metastore. The caller must be
  a metastore admin, the owner of the external metadata object, or a user that
  has the **BROWSE** privilege.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getExternalMetadataReq.Name = new(args[0])

		sdkReq := &getExternalMetadataReq
		response, err := w.ExternalMetadata.GetExternalMetadataV2(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternalmetadata.ExternalMetadataFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalMetadataOverrides {
		fn(cmd, &getExternalMetadataReq)
	}

	return cmd
}

// start list-external-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExternalMetadataOverrides []func(
	*cobra.Command,
	*ucexternalmetadata.ListExternalMetadataRequest,
)

func newListExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var listExternalMetadataReq ucexternalmetadata.ListExternalMetadataRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listExternalMetadataLimit int

	flags.IntPtrVar(cmd.Flags(), &listExternalMetadataReq.PageSize, "page-size", `Specifies the maximum number of external metadata objects to return in a single response.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listExternalMetadataLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listExternalMetadataReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-external-metadata"
	cmd.Short = `*Public Preview* List external metadata objects.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List external metadata objects.

  Gets an array of external metadata objects in the metastore. If the caller is
  the metastore admin, all external metadata objects will be retrieved.
  Otherwise, only external metadata objects that the caller has **BROWSE** on
  will be retrieved. There is no guarantee of a specific ordering of the
  elements in the array.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := ucexternalmetadata.ListExternalMetadataRequestToSDK(&listExternalMetadataReq)
		if err != nil {
			return err
		}
		if listExternalMetadataLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listExternalMetadataLimit)
		}
		if listExternalMetadataLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listExternalMetadataLimit)
		}
		response := w.ExternalMetadata.ListExternalMetadataV2Iter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucexternalmetadata.ExternalMetadataFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listExternalMetadataOverrides {
		fn(cmd, &listExternalMetadataReq)
	}

	return cmd
}

// start update-external-metadata command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExternalMetadataOverrides []func(
	*cobra.Command,
	*ucexternalmetadata.UpdateExternalMetadataRequest,
)

func newUpdateExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}
	var updateExternalMetadataReq ucexternalmetadata.UpdateExternalMetadataRequest
	updateExternalMetadataReq.ExternalMetadata = &ucexternalmetadata.ExternalMetadata{}

	var updateExternalMetadataJson flags.JsonFlag

	cmd.Flags().Var(&updateExternalMetadataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	flags.StringPtrVar(cmd.Flags(), &updateExternalMetadataReq.ExternalMetadata.Description, "description", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &updateExternalMetadataReq.ExternalMetadata.Owner, "owner", `Owner of the external metadata object.`)
	// TODO: map via StringToStringVar: properties
	flags.StringPtrVar(cmd.Flags(), &updateExternalMetadataReq.ExternalMetadata.Url, "url", `URL associated with the external metadata object.`)

	cmd.Use = "update-external-metadata NAME UPDATE_MASK SYSTEM_TYPE ENTITY_TYPE"
	cmd.Short = `*Public Preview* Update an external metadata object.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update an external metadata object.

  Updates the external metadata object that matches the supplied name. The
  caller can only update either the owner or other metadata fields in one
  request. The caller must be a metastore admin, the owner of the external
  metadata object, or a user that has the **MODIFY** privilege. If the caller is
  updating the owner, they must also have the **MANAGE** privilege.

  Arguments:
    NAME: Name of the external metadata object.
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    SYSTEM_TYPE: Type of external system.
      Supported values: [
        AMAZON_REDSHIFT,
        AZURE_SYNAPSE,
        CONFLUENT,
        DATABRICKS,
        GOOGLE_BIGQUERY,
        KAFKA,
        LOOKER,
        MICROSOFT_FABRIC,
        MICROSOFT_SQL_SERVER,
        MONGODB,
        MYSQL,
        ORACLE,
        OTHER,
        POSTGRESQL,
        POWER_BI,
        SALESFORCE,
        SAP,
        SERVICENOW,
        SNOWFLAKE,
        STREAM_NATIVE,
        TABLEAU,
        TERADATA,
        WORKDAY,
      ]
    ENTITY_TYPE: Type of entity within the external system.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'name', 'system_type', 'entity_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateExternalMetadataJson.Unmarshal(&updateExternalMetadataReq.ExternalMetadata)
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
		updateExternalMetadataReq.ExternalMetadata.Name = new(args[0])
		updateExternalMetadataReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			updateExternalMetadataReq.ExternalMetadata.SystemType = ucexternalmetadata.SystemType(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateExternalMetadataReq.ExternalMetadata.EntityType = new(args[3])
		}

		sdkReq, err := ucexternalmetadata.UpdateExternalMetadataRequestToSDK(&updateExternalMetadataReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalMetadata.UpdateExternalMetadataV2(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternalmetadata.ExternalMetadataFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateExternalMetadataOverrides {
		fn(cmd, &updateExternalMetadataReq)
	}

	return cmd
}

// end service ExternalMetadata
