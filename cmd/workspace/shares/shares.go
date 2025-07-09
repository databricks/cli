// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package shares

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "sharing",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
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
	*sharing.CreateShare,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sharing.CreateShare
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", createReq.StorageRoot, `Storage root URL for the share.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a share.`
	cmd.Long = `Create a share.
  
  Creates a new share for data objects. Data objects can be added after creation
  with **update**. The caller must be a metastore admin or have the
  **CREATE_SHARE** privilege on the metastore.

  Arguments:
    NAME: Name of the share.`

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

		response, err := w.Shares.Create(ctx, createReq)
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
	*sharing.DeleteShareRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sharing.DeleteShareRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a share.`
	cmd.Long = `Delete a share.
  
  Deletes a data object share from the metastore. The caller must be an owner of
  the share.

  Arguments:
    NAME: The name of the share.`

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

		err = w.Shares.Delete(ctx, deleteReq)
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
	*sharing.GetShareRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sharing.GetShareRequest

	cmd.Flags().BoolVar(&getReq.IncludeSharedData, "include-shared-data", getReq.IncludeSharedData, `Query for data to include in the share.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a share.`
	cmd.Long = `Get a share.
  
  Gets a data object share from the metastore. The caller must be a metastore
  admin or the owner of the share.

  Arguments:
    NAME: The name of the share.`

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

		response, err := w.Shares.Get(ctx, getReq)
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
	*sharing.ListSharesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListSharesRequest

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of shares to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List shares.`
	cmd.Long = `List shares.
  
  Gets an array of data object shares from the metastore. The caller must be a
  metastore admin or the owner of the share. There is no guarantee of a specific
  ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Shares.List(ctx, listReq)
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

// start share-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var sharePermissionsOverrides []func(
	*cobra.Command,
	*sharing.SharePermissionsRequest,
)

func newSharePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var sharePermissionsReq sharing.SharePermissionsRequest

	cmd.Flags().IntVar(&sharePermissionsReq.MaxResults, "max-results", sharePermissionsReq.MaxResults, `Maximum number of permissions to return.`)
	cmd.Flags().StringVar(&sharePermissionsReq.PageToken, "page-token", sharePermissionsReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "share-permissions NAME"
	cmd.Short = `Get permissions.`
	cmd.Long = `Get permissions.
  
  Gets the permissions for a data share from the metastore. The caller must be a
  metastore admin or the owner of the share.

  Arguments:
    NAME: The name of the share.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		sharePermissionsReq.Name = args[0]

		response, err := w.Shares.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*sharing.UpdateShare,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sharing.UpdateShare
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the share.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of share.`)
	cmd.Flags().StringVar(&updateReq.StorageRoot, "storage-root", updateReq.StorageRoot, `Storage root URL for the share.`)
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

		response, err := w.Shares.Update(ctx, updateReq)
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

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*sharing.UpdateSharePermissions,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq sharing.UpdateSharePermissions
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	cmd.Flags().BoolVar(&updatePermissionsReq.OmitPermissionsList, "omit-permissions-list", updatePermissionsReq.OmitPermissionsList, `Optional.`)

	cmd.Use = "update-permissions NAME"
	cmd.Short = `Update permissions.`
	cmd.Long = `Update permissions.
  
  Updates the permissions for a data share in the metastore. The caller must be
  a metastore admin or an owner of the share.
  
  For new recipient grants, the user must also be the recipient owner or
  metastore admin. recipient revocations do not require additional privileges.

  Arguments:
    NAME: The name of the share.`

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
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		updatePermissionsReq.Name = args[0]

		response, err := w.Shares.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
