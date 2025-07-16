// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_metadata

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
		Use:   "external-metadata",
		Short: `External Metadata objects enable customers to register and manage metadata about external systems within Unity Catalog.`,
		Long: `External Metadata objects enable customers to register and manage metadata
  about external systems within Unity Catalog.
  
  These APIs provide a standardized way to create, update, retrieve, list, and
  delete external metadata objects. Fine-grained authorization ensures that only
  users with appropriate permissions can view and manage external metadata
  objects.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*catalog.CreateExternalMetadataRequest,
)

func newCreateExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var createExternalMetadataReq catalog.CreateExternalMetadataRequest
	createExternalMetadataReq.ExternalMetadata = catalog.ExternalMetadata{}
	var createExternalMetadataJson flags.JsonFlag

	cmd.Flags().Var(&createExternalMetadataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	cmd.Flags().StringVar(&createExternalMetadataReq.ExternalMetadata.Description, "description", createExternalMetadataReq.ExternalMetadata.Description, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&createExternalMetadataReq.ExternalMetadata.Owner, "owner", createExternalMetadataReq.ExternalMetadata.Owner, `Owner of the external metadata object.`)
	// TODO: map via StringToStringVar: properties
	cmd.Flags().StringVar(&createExternalMetadataReq.ExternalMetadata.Url, "url", createExternalMetadataReq.ExternalMetadata.Url, `URL associated with the external metadata object.`)

	cmd.Use = "create-external-metadata NAME SYSTEM_TYPE ENTITY_TYPE"
	cmd.Short = `Create an external metadata object.`
	cmd.Long = `Create an external metadata object.
  
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
        TABLEAU,
        TERADATA,
        WORKDAY,
      ]
    ENTITY_TYPE: Type of entity within the external system.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'system_type', 'entity_type' in your JSON input")
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
			diags := createExternalMetadataJson.Unmarshal(&createExternalMetadataReq.ExternalMetadata)
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
			createExternalMetadataReq.ExternalMetadata.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createExternalMetadataReq.ExternalMetadata.SystemType)
			if err != nil {
				return fmt.Errorf("invalid SYSTEM_TYPE: %s", args[1])
			}
		}
		if !cmd.Flags().Changed("json") {
			createExternalMetadataReq.ExternalMetadata.EntityType = args[2]
		}

		response, err := w.ExternalMetadata.CreateExternalMetadata(ctx, createExternalMetadataReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*catalog.DeleteExternalMetadataRequest,
)

func newDeleteExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteExternalMetadataReq catalog.DeleteExternalMetadataRequest

	cmd.Use = "delete-external-metadata NAME"
	cmd.Short = `Delete an external metadata object.`
	cmd.Long = `Delete an external metadata object.
  
  Deletes the external metadata object that matches the supplied name. The
  caller must be a metastore admin, the owner of the external metadata object,
  or a user that has the **MANAGE** privilege.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteExternalMetadataReq.Name = args[0]

		err = w.ExternalMetadata.DeleteExternalMetadata(ctx, deleteExternalMetadataReq)
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
	*catalog.GetExternalMetadataRequest,
)

func newGetExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalMetadataReq catalog.GetExternalMetadataRequest

	cmd.Use = "get-external-metadata NAME"
	cmd.Short = `Get an external metadata object.`
	cmd.Long = `Get an external metadata object.
  
  Gets the specified external metadata object in a metastore. The caller must be
  a metastore admin, the owner of the external metadata object, or a user that
  has the **BROWSE** privilege.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExternalMetadataReq.Name = args[0]

		response, err := w.ExternalMetadata.GetExternalMetadata(ctx, getExternalMetadataReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*catalog.ListExternalMetadataRequest,
)

func newListExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var listExternalMetadataReq catalog.ListExternalMetadataRequest

	cmd.Flags().IntVar(&listExternalMetadataReq.PageSize, "page-size", listExternalMetadataReq.PageSize, ``)
	cmd.Flags().StringVar(&listExternalMetadataReq.PageToken, "page-token", listExternalMetadataReq.PageToken, ``)

	cmd.Use = "list-external-metadata"
	cmd.Short = `List external metadata objects.`
	cmd.Long = `List external metadata objects.
  
  Gets an array of external metadata objects in the metastore. If the caller is
  the metastore admin, all external metadata objects will be retrieved.
  Otherwise, only external metadata objects that the caller has **BROWSE** on
  will be retrieved. There is no guarantee of a specific ordering of the
  elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ExternalMetadata.ListExternalMetadata(ctx, listExternalMetadataReq)
		return cmdio.RenderIterator(ctx, response)
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
	*catalog.UpdateExternalMetadataRequest,
)

func newUpdateExternalMetadata() *cobra.Command {
	cmd := &cobra.Command{}

	var updateExternalMetadataReq catalog.UpdateExternalMetadataRequest
	updateExternalMetadataReq.ExternalMetadata = catalog.ExternalMetadata{}
	var updateExternalMetadataJson flags.JsonFlag

	cmd.Flags().Var(&updateExternalMetadataJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	cmd.Flags().StringVar(&updateExternalMetadataReq.ExternalMetadata.Description, "description", updateExternalMetadataReq.ExternalMetadata.Description, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&updateExternalMetadataReq.ExternalMetadata.Owner, "owner", updateExternalMetadataReq.ExternalMetadata.Owner, `Owner of the external metadata object.`)
	// TODO: map via StringToStringVar: properties
	cmd.Flags().StringVar(&updateExternalMetadataReq.ExternalMetadata.Url, "url", updateExternalMetadataReq.ExternalMetadata.Url, `URL associated with the external metadata object.`)

	cmd.Use = "update-external-metadata NAME SYSTEM_TYPE ENTITY_TYPE"
	cmd.Short = `Update an external metadata object.`
	cmd.Long = `Update an external metadata object.
  
  Updates the external metadata object that matches the supplied name. The
  caller can only update either the owner or other metadata fields in one
  request. The caller must be a metastore admin, the owner of the external
  metadata object, or a user that has the **MODIFY** privilege. If the caller is
  updating the owner, they must also have the **MANAGE** privilege.

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
        TABLEAU,
        TERADATA,
        WORKDAY,
      ]
    ENTITY_TYPE: Type of entity within the external system.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'name', 'system_type', 'entity_type' in your JSON input")
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
			diags := updateExternalMetadataJson.Unmarshal(&updateExternalMetadataReq.ExternalMetadata)
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
		updateExternalMetadataReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &updateExternalMetadataReq.ExternalMetadata.SystemType)
			if err != nil {
				return fmt.Errorf("invalid SYSTEM_TYPE: %s", args[1])
			}
		}
		if !cmd.Flags().Changed("json") {
			updateExternalMetadataReq.ExternalMetadata.EntityType = args[2]
		}

		response, err := w.ExternalMetadata.UpdateExternalMetadata(ctx, updateExternalMetadataReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
