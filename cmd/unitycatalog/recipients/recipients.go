// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipients

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
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

// start create command

var createReq unitycatalog.CreateRecipient
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the recipient.`)
	// TODO: any: data_recipient_global_metastore_id
	// TODO: complex arg: ip_access_list
	createCmd.Flags().StringVar(&createReq.SharingCode, "sharing-code", createReq.SharingCode, `The one-time sharing code provided by the data recipient.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share recipient.`,
	Long: `Create a share recipient.
  
  Creates a new recipient with the delta sharing authentication type in the
  Metastore. The caller must be a Metastore admin or has the CREATE_RECIPIENT
  privilege on the Metastore.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &createReq.AuthenticationType)
		if err != nil {
			return fmt.Errorf("invalid AUTHENTICATION_TYPE: %s", args[1])
		}

		response, err := w.Recipients.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteRecipientRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a share recipient.`,
	Long: `Delete a share recipient.
  
  Deletes the specified recipient from the Metastore. The caller must be the
  owner of the recipient.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, unitycatalog.ListRecipientsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		deleteReq.Name = args[0]

		err = w.Recipients.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetRecipientRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a share recipient.`,
	Long: `Get a share recipient.
  
  Gets a share recipient from the Metastore if:
  
  * the caller is the owner of the share recipient, or: * is a Metastore admin`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, unitycatalog.ListRecipientsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		getReq.Name = args[0]

		response, err := w.Recipients.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

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

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Recipients.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start rotate-token command

var rotateTokenReq unitycatalog.RotateRecipientToken

func init() {
	Cmd.AddCommand(rotateTokenCmd)
	// TODO: short flags

	rotateTokenCmd.Flags().Int64Var(&rotateTokenReq.ExistingTokenExpireInSeconds, "existing-token-expire-in-seconds", rotateTokenReq.ExistingTokenExpireInSeconds, `Required.`)

}

var rotateTokenCmd = &cobra.Command{
	Use:   "rotate-token NAME",
	Short: `Rotate a token.`,
	Long: `Rotate a token.
  
  Refreshes the specified recipient's delta sharing authentication token with
  the provided token info. The caller must be the owner of the recipient.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, unitycatalog.ListRecipientsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		rotateTokenReq.Name = args[0]

		response, err := w.Recipients.RotateToken(ctx, rotateTokenReq)
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

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions NAME",
	Short: `Get share permissions.`,
	Long: `Get share permissions.
  
  Gets the share permissions for the specified Recipient. The caller must be a
  Metastore admin or the owner of the Recipient.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, unitycatalog.ListRecipientsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		sharePermissionsReq.Name = args[0]

		response, err := w.Recipients.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateRecipient
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the recipient.`)
	// TODO: complex arg: ip_access_list
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of Recipient.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share recipient.`,
	Long: `Update a share recipient.
  
  Updates an existing recipient in the Metastore. The caller must be a Metastore
  admin or the owner of the recipient. If the recipient name will be updated,
  the user must be both a Metastore admin and the owner of the recipient.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]

		err = w.Recipients.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Recipients
