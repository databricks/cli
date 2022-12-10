// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package shares

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "shares",
	Short: `Databricks Delta Sharing: Shares REST API.`,
	Long:  `Databricks Delta Sharing: Shares REST API`,
}

// start create command

var createReq unitycatalog.CreateShare

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `comment when creating the share.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Name of the Share.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share.`,
	Long: `Create a share.
  
  Creates a new share for data objects. Data objects can be added at this time
  or after creation with **update**. The caller must be a Metastore admin or
  have the CREATE SHARE privilege on the Metastore.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteShareRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", deleteReq.Name, `The name of the share.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a share.`,
	Long: `Delete a share.
  
  Deletes a data object share from the Metastore. The caller must be an owner of
  the share.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Shares.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetShareRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeSharedData, "include-shared-data", getReq.IncludeSharedData, `Query for data to include in the share.`)
	getCmd.Flags().StringVar(&getReq.Name, "name", getReq.Name, `The name of the share.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a share.`,
	Long: `Get a share.
  
  Gets a data object share from the Metastore. The caller must be a Metastore
  admin or the owner of the share.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
  
  Gets an array of data object shares from the Metastore. The caller must be a
  Metastore admin or the owner of the share.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start share-permissions command

var sharePermissionsReq unitycatalog.SharePermissionsRequest

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags

	sharePermissionsCmd.Flags().StringVar(&sharePermissionsReq.Name, "name", sharePermissionsReq.Name, `Required.`)

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a data share from the Metastore. The caller must be a
  Metastore admin or the owner of the share.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateShare
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the share.`)
	// TODO: array: updates

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share.`,
	Long: `Update a share.
  
  Updates the share with the changes and data objects in the request. The caller
  must be the owner of the share or a Metastore admin.
  
  When the caller is a Metastore admin, only the __owner__ field can be updated.
  
  In the case that the Share name is changed, **updateShare** requires that the
  caller is both the share owner and a Metastore admin.
  
  For each table that is added through this method, the share owner must also
  have SELECT privilege on the table. This privilege must be maintained
  indefinitely for recipients to be able to access the table. Typically, you
  should use a group as the share owner.
  
  Table removals through **update** do not require additional privileges.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Shares.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update-permissions command

var updatePermissionsReq unitycatalog.UpdateSharePermissions
var updatePermissionsJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updatePermissionsCmd)
	// TODO: short flags
	updatePermissionsCmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: changes
	updatePermissionsCmd.Flags().StringVar(&updatePermissionsReq.Name, "name", updatePermissionsReq.Name, `Required.`)

}

var updatePermissionsCmd = &cobra.Command{
	Use:   "update-permissions",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a data share in the Metastore. The caller must be
  a Metastore admin or an owner of the share.
  
  For new recipient grants, the user must also be the owner of the recipients.
  recipient revocations do not require additional privileges.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updatePermissionsJson.Unmarshall(&updatePermissionsReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Shares.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Shares
