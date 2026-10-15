// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package shares

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	sharing "github.com/databricks/cli/libs/sdktypes/sharing/v1"
	sdksharing "github.com/databricks/sdk-go/sharing/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shares",
		Short: `A share is a container instantiated with :method:shares/create.`,
		Long: `A share is a container instantiated with :method:shares/create. Once created
  you can iteratively register a collection of existing data assets defined
  within the metastore using :method:shares/update. You can register data assets
  under their original name, qualified by their original schema, or provide
  alternate exposed names.`,
		GroupID: "sharing",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newListShares())
	cmd.AddCommand(newSharePermissions())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())

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
	*sharing.CreateShareRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq sharing.CreateShareRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageRoot, "storage-root", `Storage root URL for the share.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a share.`
	cmd.Long = `Create a share.

  Creates a new share for data objects. Data objects can be added after creation
  with **update**. The caller must be a metastore admin or have the
  **CREATE_SHARE** privilege on the metastore.

  Arguments:
    NAME: Name of the share.`

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

		sdkReq, err := sharing.CreateShareRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.CreateShare(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ShareInfoFromSDK(response)
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
	*sdksharing.DeleteShareRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdksharing.DeleteShareRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a share.`
	cmd.Long = `Delete a share.

  Deletes a data object share from the metastore. The caller must be an owner of
  the share.

  Arguments:
    NAME: The name of the share.`

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

		deleteReq.Name = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Sharing.DeleteShare(ctx, *sdkReq)
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
	*sharing.GetShareRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sharing.GetShareRequest

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeSharedData, "include-shared-data", `Query for data to include in the share.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a share.`
	cmd.Long = `Get a share.

  Gets a data object share from the metastore. The caller must have the
  USE_SHARE privilege on the metastore or be the owner of the share.

  Arguments:
    NAME: The name of the share.`

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

		getReq.Name = new(args[0])

		sdkReq, err := sharing.GetShareRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.GetShare(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ShareInfoFromSDK(response)
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

// start list-shares command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSharesOverrides []func(
	*cobra.Command,
	*sharing.ListSharesRequest,
)

func newListShares() *cobra.Command {
	cmd := &cobra.Command{}
	var listSharesReq sharing.ListSharesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSharesLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSharesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSharesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listSharesReq.MaxResults, "max-results", `Maximum number of shares to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list-shares"
	cmd.Short = `List shares.`
	cmd.Long = `List shares.

  Gets an array of data object shares from the metastore. If the caller has the
  USE_SHARE privilege on the metastore, all shares are returned. Otherwise, only
  shares owned by the caller are returned. There is no guarantee of a specific
  ordering of the elements in the array.`

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

		sdkReq, err := sharing.ListSharesRequestToSDK(&listSharesReq)
		if err != nil {
			return err
		}
		if listSharesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSharesLimit)
		}
		if listSharesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSharesLimit)
		}
		response := w.Sharing.ListSharesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, sharing.ShareInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSharesOverrides {
		fn(cmd, &listSharesReq)
	}

	return cmd
}

// start share-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var sharePermissionsOverrides []func(
	*cobra.Command,
	*sharing.ListSharePermissionsRequest,
)

func newSharePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var sharePermissionsReq sharing.ListSharePermissionsRequest

	flags.IntPtrVar(cmd.Flags(), &sharePermissionsReq.MaxResults, "max-results", `Maximum number of permissions to return.`)
	flags.StringPtrVar(cmd.Flags(), &sharePermissionsReq.PageToken, "page-token", `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "share-permissions NAME"
	cmd.Short = `Get permissions.`
	cmd.Long = `Get permissions.

  Gets the permissions for a data share from the metastore. The caller must have
  the USE_SHARE privilege on the metastore or be the owner of the share.

  Arguments:
    NAME: The name of the Recipient.`

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

		sharePermissionsReq.Name = new(args[0])

		sdkReq, err := sharing.ListSharePermissionsRequestToSDK(&sharePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.ListSharePermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.GetSharePermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range sharePermissionsOverrides {
		fn(cmd, &sharePermissionsReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sharing.UpdateShareRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq sharing.UpdateShareRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the share.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `Username of current owner of share.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.StorageRoot, "storage-root", `Storage root URL for the share.`)
	// TODO: array: updates

	cmd.Use = "update NAME"
	cmd.Short = `Update a share.`
	cmd.Long = `Update a share.

  Updates the share with the changes and data objects in the request. The caller
  must be the owner of the share or a metastore admin.

  When the caller is a metastore admin, only the __owner__ field can be updated.

  In the case the share name is changed, **updateShare** requires that the
  caller is the owner of the share and has the CREATE_SHARE privilege.

  If there are notebook files in the share, the __storage_root__ field cannot be
  updated.

  For each table that is added through this method, the share owner must also
  have **SELECT** privilege on the table. This privilege must be maintained
  indefinitely for recipients to be able to access the table. Typically, you
  should use a group as the share owner.

  Table removals through **update** do not require additional privileges.

  Arguments:
    NAME: The name of the share.`

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

		sdkReq, err := sharing.UpdateShareRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.UpdateShare(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ShareInfoFromSDK(response)
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

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*sharing.UpdateSharePermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq sharing.UpdateSharePermissionsRequest

	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	flags.BoolPtrVar(cmd.Flags(), &updatePermissionsReq.OmitPermissionsList, "omit-permissions-list", `Optional.`)

	cmd.Use = "update-permissions NAME"
	cmd.Short = `Update permissions.`
	cmd.Long = `Update permissions.

  Updates the permissions for a data share in the metastore. The caller must
  have both the USE_SHARE and SET_SHARE_PERMISSION privileges on the metastore,
  or be the owner of the share.

  For new recipient grants, the user must also be the owner of the recipients.
  recipient revocations do not require additional privileges.

  Arguments:
    NAME: The name of the share.`

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
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		updatePermissionsReq.Name = new(args[0])

		sdkReq, err := sharing.UpdateSharePermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.UpdateSharePermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.UpdateSharePermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service Shares
