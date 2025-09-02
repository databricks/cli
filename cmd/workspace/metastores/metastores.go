// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

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
		Use:   "metastores",
		Short: `A metastore is the top-level container of objects in Unity Catalog.`,
		Long: `A metastore is the top-level container of objects in Unity Catalog. It stores
  data assets (tables and views) and the permissions that govern access to them.
  Databricks account admins can create metastores and assign them to Databricks
  workspaces to control which workloads use each metastore. For a workspace to
  use Unity Catalog, it must have a Unity Catalog metastore attached.
  
  Each metastore is configured with a root storage location in a cloud storage
  account. This storage location is used for metadata and managed tables data.
  
  NOTE: This metastore is distinct from the metastore included in Databricks
  workspaces created before Unity Catalog was released. If your workspace
  includes a legacy Hive metastore, the data in that metastore is available in a
  catalog named hive_metastore.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newAssign())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCurrent())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSummary())
	cmd.AddCommand(newUnassign())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdateAssignment())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start assign command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var assignOverrides []func(
	*cobra.Command,
	*catalog.CreateMetastoreAssignment,
)

func newAssign() *cobra.Command {
	cmd := &cobra.Command{}

	var assignReq catalog.CreateMetastoreAssignment
	var assignJson flags.JsonFlag

	cmd.Flags().Var(&assignJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "assign WORKSPACE_ID METASTORE_ID DEFAULT_CATALOG_NAME"
	cmd.Short = `Create an assignment.`
	cmd.Long = `Create an assignment.
  
  Creates a new metastore assignment. If an assignment for the same
  __workspace_id__ exists, it will be overwritten by the new __metastore_id__
  and __default_catalog_name__. The caller must be an account admin.

  Arguments:
    WORKSPACE_ID: A workspace ID.
    METASTORE_ID: The unique ID of the metastore.
    DEFAULT_CATALOG_NAME: The name of the default catalog in the metastore. This field is
      deprecated. Please use "Default Namespace API" to configure the default
      catalog for a Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only WORKSPACE_ID as positional arguments. Provide 'metastore_id', 'default_catalog_name' in your JSON input")
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
			diags := assignJson.Unmarshal(&assignReq)
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
		_, err = fmt.Sscan(args[0], &assignReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		if !cmd.Flags().Changed("json") {
			assignReq.MetastoreId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			assignReq.DefaultCatalogName = args[2]
		}

		err = w.Metastores.Assign(ctx, assignReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range assignOverrides {
		fn(cmd, &assignReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*catalog.CreateMetastore,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateMetastore
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Region, "region", createReq.Region, `Cloud region which the metastore serves (e.g., us-west-2, westus).`)
	cmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", createReq.StorageRoot, `The storage root URL for metastore.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a metastore.`
	cmd.Long = `Create a metastore.
  
  Creates a new metastore based on a provided name and optional storage root
  path. By default (if the __owner__ field is not set), the owner of the new
  metastore is the user calling the __createMetastore__ API. If the __owner__
  field is set to the empty string (**""**), the ownership is assigned to the
  System User instead.

  Arguments:
    NAME: The user-specified name of the metastore.`

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

		response, err := w.Metastores.Create(ctx, createReq)
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

// start current command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var currentOverrides []func(
	*cobra.Command,
)

func newCurrent() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "current"
	cmd.Short = `Get metastore assignment for workspace.`
	cmd.Long = `Get metastore assignment for workspace.
  
  Gets the metastore assignment for the workspace being accessed.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Metastores.Current(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range currentOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteMetastoreRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteMetastoreRequest

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the metastore is not empty.`)

	cmd.Use = "delete ID"
	cmd.Short = `Delete a metastore.`
	cmd.Long = `Delete a metastore.
  
  Deletes a metastore. The caller must be a metastore admin.

  Arguments:
    ID: Unique ID of the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Id = args[0]

		err = w.Metastores.Delete(ctx, deleteReq)
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
	*catalog.GetMetastoreRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetMetastoreRequest

	cmd.Use = "get ID"
	cmd.Short = `Get a metastore.`
	cmd.Long = `Get a metastore.
  
  Gets a metastore that matches the supplied ID. The caller must be a metastore
  admin to retrieve this info.

  Arguments:
    ID: Unique ID of the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Id = args[0]

		response, err := w.Metastores.Get(ctx, getReq)
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
	*catalog.ListMetastoresRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListMetastoresRequest

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of metastores to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List metastores.`
	cmd.Long = `List metastores.
  
  Gets an array of the available metastores (as __MetastoreInfo__ objects). The
  caller must be an admin to retrieve this info. There is no guarantee of a
  specific ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Metastores.List(ctx, listReq)
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

// start summary command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var summaryOverrides []func(
	*cobra.Command,
)

func newSummary() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "summary"
	cmd.Short = `Get a metastore summary.`
	cmd.Long = `Get a metastore summary.
  
  Gets information about a metastore. This summary includes the storage
  credential, the cloud vendor, the cloud region, and the global metastore ID.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Metastores.Summary(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range summaryOverrides {
		fn(cmd)
	}

	return cmd
}

// start unassign command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var unassignOverrides []func(
	*cobra.Command,
	*catalog.UnassignRequest,
)

func newUnassign() *cobra.Command {
	cmd := &cobra.Command{}

	var unassignReq catalog.UnassignRequest

	cmd.Use = "unassign WORKSPACE_ID METASTORE_ID"
	cmd.Short = `Delete an assignment.`
	cmd.Long = `Delete an assignment.
  
  Deletes a metastore assignment. The caller must be an account administrator.

  Arguments:
    WORKSPACE_ID: A workspace ID.
    METASTORE_ID: Query for the ID of the metastore to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &unassignReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		unassignReq.MetastoreId = args[1]

		err = w.Metastores.Unassign(ctx, unassignReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range unassignOverrides {
		fn(cmd, &unassignReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateMetastore,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateMetastore
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.DeltaSharingOrganizationName, "delta-sharing-organization-name", updateReq.DeltaSharingOrganizationName, `The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.`)
	cmd.Flags().Int64Var(&updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, `The lifetime of delta sharing recipient token in seconds.`)
	cmd.Flags().Var(&updateReq.DeltaSharingScope, "delta-sharing-scope", `The scope of Delta Sharing enabled for the metastore. Supported values: [INTERNAL, INTERNAL_AND_EXTERNAL]`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the metastore.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the metastore.`)
	cmd.Flags().StringVar(&updateReq.PrivilegeModelVersion, "privilege-model-version", updateReq.PrivilegeModelVersion, `Privilege model version of the metastore, of the form major.minor (e.g., 1.0).`)
	cmd.Flags().StringVar(&updateReq.StorageRootCredentialId, "storage-root-credential-id", updateReq.StorageRootCredentialId, `UUID of storage credential to access the metastore storage_root.`)

	cmd.Use = "update ID"
	cmd.Short = `Update a metastore.`
	cmd.Long = `Update a metastore.
  
  Updates information for a specific metastore. The caller must be a metastore
  admin. If the __owner__ field is set to the empty string (**""**), the
  ownership is updated to the System User.

  Arguments:
    ID: Unique ID of the metastore.`

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
		updateReq.Id = args[0]

		response, err := w.Metastores.Update(ctx, updateReq)
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

// start update-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAssignmentOverrides []func(
	*cobra.Command,
	*catalog.UpdateMetastoreAssignment,
)

func newUpdateAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateAssignmentReq catalog.UpdateMetastoreAssignment
	var updateAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateAssignmentReq.DefaultCatalogName, "default-catalog-name", updateAssignmentReq.DefaultCatalogName, `The name of the default catalog in the metastore.`)
	cmd.Flags().StringVar(&updateAssignmentReq.MetastoreId, "metastore-id", updateAssignmentReq.MetastoreId, `The unique ID of the metastore.`)

	cmd.Use = "update-assignment WORKSPACE_ID"
	cmd.Short = `Update an assignment.`
	cmd.Long = `Update an assignment.
  
  Updates a metastore assignment. This operation can be used to update
  __metastore_id__ or __default_catalog_name__ for a specified Workspace, if the
  Workspace is already assigned a metastore. The caller must be an account admin
  to update __metastore_id__; otherwise, the caller can be a Workspace admin.

  Arguments:
    WORKSPACE_ID: A workspace ID.`

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
			diags := updateAssignmentJson.Unmarshal(&updateAssignmentReq)
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
		_, err = fmt.Sscan(args[0], &updateAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		err = w.Metastores.UpdateAssignment(ctx, updateAssignmentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAssignmentOverrides {
		fn(cmd, &updateAssignmentReq)
	}

	return cmd
}

// end service Metastores
