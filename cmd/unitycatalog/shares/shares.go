package shares

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "shares",
	Short: `Databricks Delta Sharing: Shares REST API.`,
}

var createReq unitycatalog.CreateShare

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `[Create: OPT] comment when creating the share.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", 0, `[Create:IGN] Time at which this Share was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", "", `[Create:IGN] Username of Share creator.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `[Create:REQ] Name of the Share.`)
	// TODO: array: objects

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share.`,
	Long: `Create a share.
  
  Creates a new share for data objects. Data objects can be added at this time
  or after creation with **update**. The caller must be a Metastore admin or
  have the CREATE SHARE privilege on the Metastore.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq unitycatalog.DeleteShareRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", "", `The name of the share.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a share.`,
	Long: `Delete a share.
  
  Deletes a data object share from the Metastore. The caller must be an owner of
  the share.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Shares.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq unitycatalog.GetShareRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeSharedData, "include-shared-data", false, `Query for data to include in the share.`)
	getCmd.Flags().StringVar(&getReq.Name, "name", "", `The name of the share.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a share.`,
	Long: `Get a share.
  
  Gets a data object share from the Metastore. The caller must be a Metastore
  admin or the owner of the share.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List shares.`,
	Long: `List shares.
  
  Gets an array of data object shares from the Metastore. The caller must be a
  Metastore admin or the owner of the share.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.ListAll(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var sharePermissionsReq unitycatalog.SharePermissionsRequest

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags

	sharePermissionsCmd.Flags().StringVar(&sharePermissionsReq.Name, "name", "", `Required.`)

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions",
	Short: `Get permissions.`,
	Long: `Get permissions.
  
  Gets the permissions for a data share from the Metastore. The caller must be a
  Metastore admin or the owner of the share.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Shares.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var updateReq unitycatalog.UpdateShare

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `The name of the share.`)
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

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Shares.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updatePermissionsReq unitycatalog.UpdateSharePermissions

func init() {
	Cmd.AddCommand(updatePermissionsCmd)
	// TODO: short flags

	// TODO: array: changes
	updatePermissionsCmd.Flags().StringVar(&updatePermissionsReq.Name, "name", "", `Required.`)

}

var updatePermissionsCmd = &cobra.Command{
	Use:   "update-permissions",
	Short: `Update permissions.`,
	Long: `Update permissions.
  
  Updates the permissions for a data share in the Metastore. The caller must be
  a Metastore admin or an owner of the share.
  
  For new recipient grants, the user must also be the owner of the recipients.
  recipient revocations do not require additional privileges.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Shares.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}

		return nil
	},
}
