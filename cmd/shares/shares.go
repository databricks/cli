// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package shares

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "shares",
	Short: `Databricks Shares REST API.`,
	Long:  `Databricks Shares REST API`,
}

// start create command

var createReq sharing.CreateShare

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME",
	Short: `Create a share.`,
	Long: `Create a share.
  
  Creates a new share for data objects. Data objects can be added after creation
  with **update**. The caller must be a metastore admin or have the
  **CREATE_SHARE** privilege on the metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		createReq.Name = args[0]

		response, err := w.Shares.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq sharing.DeleteShareRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a share.`,
	Long: `Delete a share.
  
  Deletes a data object share from the metastore. The caller must be an owner of
  the share.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		deleteReq.Name = args[0]

		err = w.Shares.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sharing.GetShareRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeSharedData, "include-shared-data", getReq.IncludeSharedData, `Query for data to include in the share.`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a share.`,
	Long: `Get a share.
  
  Gets a data object share from the metastore. The caller must be a metastore
  admin or the owner of the share.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getReq.Name = args[0]

		response, err := w.Shares.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List shares.`,
	Long: `List shares.
  
  Gets an array of data object shares from the metastore. The caller must be a
  metastore admin or the owner of the share. There is no guarantee of a specific
  ordering of the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Shares.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start share-permissions command

var sharePermissionsReq sharing.SharePermissionsRequest

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions NAME",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a data share from the metastore. The caller must be a
  metastore admin or the owner of the share.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		sharePermissionsReq.Name = args[0]

		response, err := w.Shares.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq sharing.UpdateShare
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of the share.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of share.`)
	// TODO: array: updates

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share.`,
	Long: `Update a share.
  
  Updates the share with the changes and data objects in the request. The caller
  must be the owner of the share or a metastore admin.
  
  When the caller is a metastore admin, only the __owner__ field can be updated.
  
  In the case that the share name is changed, **updateShare** requires that the
  caller is both the share owner and a metastore admin.
  
  For each table that is added through this method, the share owner must also
  have **SELECT** privilege on the table. This privilege must be maintained
  indefinitely for recipients to be able to access the table. Typically, you
  should use a group as the share owner.
  
  Table removals through **update** do not require additional privileges.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]

		response, err := w.Shares.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update-permissions command

var updatePermissionsReq sharing.UpdateSharePermissions
var updatePermissionsJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updatePermissionsCmd)
	// TODO: short flags
	updatePermissionsCmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes

}

var updatePermissionsCmd = &cobra.Command{
	Use:   "update-permissions",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a data share in the metastore. The caller must be
  a metastore admin or an owner of the share.
  
  For new recipient grants, the user must also be the owner of the recipients.
  recipient revocations do not require additional privileges.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updatePermissionsJson.Unmarshall(&updatePermissionsReq)
		if err != nil {
			return err
		}
		updatePermissionsReq.Name = args[0]

		err = w.Shares.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Shares
