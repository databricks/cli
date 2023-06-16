// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipients

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipients",
	Short: `Databricks Recipients REST API.`,
	Long:  `Databricks Recipients REST API`,
	Annotations: map[string]string{
		"package": "sharing",
	},
}

// start create command

var createReq sharing.CreateRecipient
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the recipient.`)
	// TODO: any: data_recipient_global_metastore_id
	// TODO: complex arg: ip_access_list
	createCmd.Flags().StringVar(&createReq.Owner, "owner", createReq.Owner, `Username of the recipient owner.`)
	// TODO: complex arg: properties_kvpairs
	createCmd.Flags().StringVar(&createReq.SharingCode, "sharing-code", createReq.SharingCode, `The one-time sharing code provided by the data recipient.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME AUTHENTICATION_TYPE",
	Short: `Create a share recipient.`,
	Long: `Create a share recipient.
  
  Creates a new recipient with the delta sharing authentication type in the
  metastore. The caller must be a metastore admin or has the
  **CREATE_RECIPIENT** privilege on the metastore.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
			_, err = fmt.Sscan(args[1], &createReq.AuthenticationType)
			if err != nil {
				return fmt.Errorf("invalid AUTHENTICATION_TYPE: %s", args[1])
			}
		}

		response, err := w.Recipients.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command

var deleteReq sharing.DeleteRecipientRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a share recipient.`,
	Long: `Delete a share recipient.
  
  Deletes the specified recipient from the metastore. The caller must be the
  owner of the recipient.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Recipients drop-down."
				names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, sharing.ListRecipientsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Recipients drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the recipient")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the recipient")
			}
			deleteReq.Name = args[0]
		}

		err = w.Recipients.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command

var getReq sharing.GetRecipientRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a share recipient.`,
	Long: `Get a share recipient.
  
  Gets a share recipient from the metastore if:
  
  * the caller is the owner of the share recipient, or: * is a metastore admin`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Recipients drop-down."
				names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, sharing.ListRecipientsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Recipients drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the recipient")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the recipient")
			}
			getReq.Name = args[0]
		}

		response, err := w.Recipients.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

var listReq sharing.ListRecipientsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().StringVar(&listReq.DataRecipientGlobalMetastoreId, "data-recipient-global-metastore-id", listReq.DataRecipientGlobalMetastoreId, `If not provided, all recipients will be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List share recipients.`,
	Long: `List share recipients.
  
  Gets an array of all share recipients within the current metastore where:
  
  * the caller is a metastore admin, or * the caller is the owner. There is no
  guarantee of a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Recipients.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start rotate-token command

var rotateTokenReq sharing.RotateRecipientToken
var rotateTokenJson flags.JsonFlag

func init() {
	Cmd.AddCommand(rotateTokenCmd)
	// TODO: short flags
	rotateTokenCmd.Flags().Var(&rotateTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var rotateTokenCmd = &cobra.Command{
	Use:   "rotate-token EXISTING_TOKEN_EXPIRE_IN_SECONDS NAME",
	Short: `Rotate a token.`,
	Long: `Rotate a token.
  
  Refreshes the specified recipient's delta sharing authentication token with
  the provided token info. The caller must be the owner of the recipient.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = rotateTokenJson.Unmarshal(&rotateTokenReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &rotateTokenReq.ExistingTokenExpireInSeconds)
			if err != nil {
				return fmt.Errorf("invalid EXISTING_TOKEN_EXPIRE_IN_SECONDS: %s", args[0])
			}
			rotateTokenReq.Name = args[1]
		}

		response, err := w.Recipients.RotateToken(ctx, rotateTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start share-permissions command

var sharePermissionsReq sharing.SharePermissionsRequest
var sharePermissionsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags
	sharePermissionsCmd.Flags().Var(&sharePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions NAME",
	Short: `Get recipient share permissions.`,
	Long: `Get recipient share permissions.
  
  Gets the share permissions for the specified Recipient. The caller must be a
  metastore admin or the owner of the Recipient.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = sharePermissionsJson.Unmarshal(&sharePermissionsReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Recipients drop-down."
				names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, sharing.ListRecipientsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Recipients drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The name of the Recipient")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the name of the recipient")
			}
			sharePermissionsReq.Name = args[0]
		}

		response, err := w.Recipients.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command

var updateReq sharing.UpdateRecipient
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the recipient.`)
	// TODO: complex arg: ip_access_list
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of Recipient.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of the recipient owner.`)
	// TODO: complex arg: properties_kvpairs

}

var updateCmd = &cobra.Command{
	Use:   "update NAME",
	Short: `Update a share recipient.`,
	Long: `Update a share recipient.
  
  Updates an existing recipient in the metastore. The caller must be a metastore
  admin or the owner of the recipient. If the recipient name will be updated,
  the user must be both a metastore admin and the owner of the recipient.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Recipients drop-down."
				names, err := w.Recipients.RecipientInfoNameToMetastoreIdMap(ctx, sharing.ListRecipientsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Recipients drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of Recipient")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of recipient")
			}
			updateReq.Name = args[0]
		}

		err = w.Recipients.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service Recipients
