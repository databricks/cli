// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucmetastores "github.com/databricks/cli/libs/sdktypes/uc/metastores/v1"
	sdkucmetastores "github.com/databricks/sdk-go/uc/metastores/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*ucmetastores.CreateMetastoreAssignmentRequest,
)

func newAssign() *cobra.Command {
	cmd := &cobra.Command{}
	var assignReq ucmetastores.CreateMetastoreAssignmentRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WORKSPACE_ID as positional arguments. Provide 'metastore_id', 'default_catalog_name' in your JSON input")
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
			diags := assignJson.Unmarshal(&assignReq)
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
		var assignWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &assignWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		assignReq.WorkspaceId = &assignWorkspaceIdArg
		if !cmd.Flags().Changed("json") {
			assignReq.MetastoreId = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			assignReq.DefaultCatalogName = new(args[2])
		}

		sdkReq, err := ucmetastores.CreateMetastoreAssignmentRequestToSDK(&assignReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.CreateMetastoreAssignment(ctx, *sdkReq)
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
	*ucmetastores.CreateMetastoreRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucmetastores.CreateMetastoreRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.ExternalAccessEnabled, "external-access-enabled", `Whether to allow non-DBR clients to directly access entities under the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Region, "region", `Cloud region which the metastore serves (e.g., us-west-2, westus).`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageRoot, "storage-root", `The storage root URL for metastore.`)

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

		sdkReq, err := ucmetastores.CreateMetastoreRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.CreateMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.MetastoreInfoFromSDK(response)
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkucmetastores.GetCurrentMetastoreAssignmentRequest{}

		response, err := w.Metastores.GetCurrentMetastoreAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.MetastoreAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucmetastores.DeleteMetastoreRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq ucmetastores.DeleteMetastoreRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force deletion even if the metastore is not empty.`)

	cmd.Use = "delete ID"
	cmd.Short = `Delete a metastore.`
	cmd.Long = `Delete a metastore.

  Deletes a metastore. The caller must be a metastore admin.

  Arguments:
    ID: Unique ID of the metastore.`

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

		deleteReq.Id = new(args[0])

		sdkReq, err := ucmetastores.DeleteMetastoreRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.DeleteMetastore(ctx, *sdkReq)
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
	*sdkucmetastores.GetMetastoreRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucmetastores.GetMetastoreRequest

	cmd.Use = "get ID"
	cmd.Short = `Get a metastore.`
	cmd.Long = `Get a metastore.

  Gets a metastore that matches the supplied ID. The caller must be a metastore
  admin to retrieve this info.

  Arguments:
    ID: Unique ID of the metastore.`

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

		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Metastores.GetMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.MetastoreInfoFromSDK(response)
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
	*ucmetastores.ListMetastoresRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucmetastores.ListMetastoresRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of metastores to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List metastores.`
	cmd.Long = `List metastores.

  Gets an array of the available metastores (as __MetastoreInfo__ objects). The
  caller must be an admin to retrieve this info. There is no guarantee of a
  specific ordering of the elements in the array.

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

		sdkReq, err := ucmetastores.ListMetastoresRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Metastores.ListMetastoresIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucmetastores.MetastoreInfoFromSDK))
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkucmetastores.GetMetastoreSummaryRequest{}

		response, err := w.Metastores.GetMetastoreSummary(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.GetMetastoreSummaryResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucmetastores.DeleteMetastoreAssignmentRequest,
)

func newUnassign() *cobra.Command {
	cmd := &cobra.Command{}
	var unassignReq ucmetastores.DeleteMetastoreAssignmentRequest

	cmd.Use = "unassign WORKSPACE_ID METASTORE_ID"
	cmd.Short = `Delete an assignment.`
	cmd.Long = `Delete an assignment.

  Deletes a metastore assignment. The caller must be an account administrator.

  Arguments:
    WORKSPACE_ID: A workspace ID.
    METASTORE_ID: Query for the ID of the metastore to delete.`

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

		var unassignWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &unassignWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		unassignReq.WorkspaceId = &unassignWorkspaceIdArg
		unassignReq.MetastoreId = new(args[1])

		sdkReq, err := ucmetastores.DeleteMetastoreAssignmentRequestToSDK(&unassignReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.DeleteMetastoreAssignment(ctx, *sdkReq)
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
	*ucmetastores.UpdateMetastoreRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucmetastores.UpdateMetastoreRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.DeltaSharingOrganizationName, "delta-sharing-organization-name", `The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", `The lifetime of delta sharing recipient token in seconds.`)
	flags.EnumVar(cmd.Flags(), &updateReq.DeltaSharingScope, "delta-sharing-scope", `The scope of Delta Sharing enabled for the metastore. Supported values: [INTERNAL, INTERNAL_AND_EXTERNAL]`, "DeltaSharingScopeEnum", "INTERNAL", "INTERNAL_AND_EXTERNAL")
	flags.BoolPtrVar(cmd.Flags(), &updateReq.ExternalAccessEnabled, "external-access-enabled", `Whether to allow non-DBR clients to directly access entities under the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `The owner of the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.PrivilegeModelVersion, "privilege-model-version", `Privilege model version of the metastore, of the form major.minor (e.g., 1.0).`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.StorageRootCredentialId, "storage-root-credential-id", `UUID of storage credential to access the metastore storage_root.`)

	cmd.Use = "update ID"
	cmd.Short = `Update a metastore.`
	cmd.Long = `Update a metastore.

  Updates information for a specific metastore. The caller must be a metastore
  admin. If the __owner__ field is set to the empty string (**""**), the
  ownership is updated to the System User.

  Arguments:
    ID: Unique ID of the metastore.`

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
		updateReq.Id = new(args[0])

		sdkReq, err := ucmetastores.UpdateMetastoreRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.UpdateMetastore(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucmetastores.MetastoreInfoFromSDK(response)
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

// start update-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAssignmentOverrides []func(
	*cobra.Command,
	*ucmetastores.UpdateMetastoreAssignmentRequest,
)

func newUpdateAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var updateAssignmentReq ucmetastores.UpdateMetastoreAssignmentRequest

	var updateAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateAssignmentReq.DefaultCatalogName, "default-catalog-name", `The name of the default catalog in the metastore.`)
	flags.StringPtrVar(cmd.Flags(), &updateAssignmentReq.MetastoreId, "metastore-id", `The unique ID of the metastore.`)

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
			diags := updateAssignmentJson.Unmarshal(&updateAssignmentReq)
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
		var updateAssignmentWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &updateAssignmentWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		updateAssignmentReq.WorkspaceId = &updateAssignmentWorkspaceIdArg

		sdkReq, err := ucmetastores.UpdateMetastoreAssignmentRequestToSDK(&updateAssignmentReq)
		if err != nil {
			return err
		}
		response, err := w.Metastores.UpdateMetastoreAssignment(ctx, *sdkReq)
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
	for _, fn := range updateAssignmentOverrides {
		fn(cmd, &updateAssignmentReq)
	}

	return cmd
}

// end service Metastores
