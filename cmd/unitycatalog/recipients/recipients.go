package recipients

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipients",
	Short: `Databricks Delta Sharing: Recipients REST API.`,
	Long:  `Databricks Delta Sharing: Recipients REST API`,
}

var createReq unitycatalog.CreateRecipient

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Activated, "activated", createReq.Activated, `[Create:IGN,Update:IGN] A boolean status field showing whether the Recipient's activation URL has been exercised or not.`)
	createCmd.Flags().StringVar(&createReq.ActivationUrl, "activation-url", createReq.ActivationUrl, `[Create:IGN,Update:IGN] Full activation url to retrieve the access token.`)
	createCmd.Flags().Var(&createReq.AuthenticationType, "authentication-type", `[Create:REQ,Update:IGN] The delta sharing authentication type.`)
	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `[Create:OPT,Update:OPT] Description about the recipient.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", createReq.CreatedAt, `[Create:IGN,Update:IGN] Time at which this recipient was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", createReq.CreatedBy, `[Create:IGN,Update:IGN] Username of recipient creator.`)
	// TODO: complex arg: ip_access_list
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `[Create:REQ,Update:OPT] Name of Recipient.`)
	createCmd.Flags().StringVar(&createReq.SharingCode, "sharing-code", createReq.SharingCode, `[Create:OPT,Update:IGN] The one-time sharing code provided by the data recipient.`)
	// TODO: array: tokens
	createCmd.Flags().Int64Var(&createReq.UpdatedAt, "updated-at", createReq.UpdatedAt, `[Create:IGN,Update:IGN] Time at which the recipient was updated, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.UpdatedBy, "updated-by", createReq.UpdatedBy, `[Create:IGN,Update:IGN] Username of recipient updater.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share recipient.`,
	Long: `Create a share recipient.
  
  Creates a new recipient with the delta sharing authentication type in the
  Metastore. The caller must be a Metastore admin or has the CREATE RECIPIENT
  privilege on the Metastore.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Recipients.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq unitycatalog.DeleteRecipientRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", deleteReq.Name, `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a share recipient.`,
	Long: `Delete a share recipient.
  
  Deletes the specified recipient from the Metastore. The caller must be the
  owner of the recipient.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Recipients.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getReq unitycatalog.GetRecipientRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Name, "name", getReq.Name, `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a share recipient.`,
	Long: `Get a share recipient.
  
  Gets a share recipient from the Metastore if:
  
  * the caller is the owner of the share recipient, or: * is a Metastore admin`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Recipients.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listReq unitycatalog.ListRecipientsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.DataRecipientGlobalMetastoreId, "data-recipient-global-metastore-id", listReq.DataRecipientGlobalMetastoreId, `If not provided, all recipients will be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List share recipients.`,
	Long: `List share recipients.
  
  Gets an array of all share recipients within the current Metastore where:
  
  * the caller is a Metastore admin, or * the caller is the owner.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Recipients.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var rotateTokenReq unitycatalog.RotateRecipientToken

func init() {
	Cmd.AddCommand(rotateTokenCmd)
	// TODO: short flags

	rotateTokenCmd.Flags().Int64Var(&rotateTokenReq.ExistingTokenExpireInSeconds, "existing-token-expire-in-seconds", rotateTokenReq.ExistingTokenExpireInSeconds, `Required.`)
	rotateTokenCmd.Flags().StringVar(&rotateTokenReq.Name, "name", rotateTokenReq.Name, `Required.`)

}

var rotateTokenCmd = &cobra.Command{
	Use:   "rotate-token",
	Short: `Rotate a token.`,
	Long: `Rotate a token.
  
  Refreshes the specified recipient's delta sharing authentication token with
  the provided token info. The caller must be the owner of the recipient.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Recipients.RotateToken(ctx, rotateTokenReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var sharePermissionsReq unitycatalog.SharePermissionsRequest

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags

	sharePermissionsCmd.Flags().StringVar(&sharePermissionsReq.Name, "name", sharePermissionsReq.Name, `Required.`)

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions",
	Short: `Get share permissions.`,
	Long: `Get share permissions.
  
  Gets the share permissions for the specified Recipient. The caller must be a
  Metastore admin or the owner of the Recipient.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Recipients.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var updateReq unitycatalog.UpdateRecipient

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Activated, "activated", updateReq.Activated, `[Create:IGN,Update:IGN] A boolean status field showing whether the Recipient's activation URL has been exercised or not.`)
	updateCmd.Flags().StringVar(&updateReq.ActivationUrl, "activation-url", updateReq.ActivationUrl, `[Create:IGN,Update:IGN] Full activation url to retrieve the access token.`)
	updateCmd.Flags().Var(&updateReq.AuthenticationType, "authentication-type", `[Create:REQ,Update:IGN] The delta sharing authentication type.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `[Create:OPT,Update:OPT] Description about the recipient.`)
	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", updateReq.CreatedAt, `[Create:IGN,Update:IGN] Time at which this recipient was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", updateReq.CreatedBy, `[Create:IGN,Update:IGN] Username of recipient creator.`)
	// TODO: complex arg: ip_access_list
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `[Create:REQ,Update:OPT] Name of Recipient.`)
	updateCmd.Flags().StringVar(&updateReq.SharingCode, "sharing-code", updateReq.SharingCode, `[Create:OPT,Update:IGN] The one-time sharing code provided by the data recipient.`)
	// TODO: array: tokens
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", updateReq.UpdatedAt, `[Create:IGN,Update:IGN] Time at which the recipient was updated, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", updateReq.UpdatedBy, `[Create:IGN,Update:IGN] Username of recipient updater.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share recipient.`,
	Long: `Update a share recipient.
  
  Updates an existing recipient in the Metastore. The caller must be a Metastore
  admin or the owner of the recipient. If the recipient name will be updated,
  the user must be both a Metastore admin and the owner of the recipient.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Recipients.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Recipients
