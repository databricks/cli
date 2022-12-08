package recipients

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipients",
	Short: `Databricks Delta Sharing: Recipients REST API.`,
}

var createReq unitycatalog.CreateRecipient

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Activated, "activated", false, `[Create:IGN,Update:IGN] A boolean status field showing whether the Recipient's activation URL has been exercised or not.`)
	createCmd.Flags().StringVar(&createReq.ActivationUrl, "activation-url", "", `[Create:IGN,Update:IGN] Full activation url to retrieve the access token.`)
	createCmd.Flags().Var(&createReq.AuthenticationType, "authentication-type", `[Create:REQ,Update:IGN] The delta sharing authentication type.`)
	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `[Create:OPT,Update:OPT] Description about the recipient.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", 0, `[Create:IGN,Update:IGN] Time at which this recipient was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", "", `[Create:IGN,Update:IGN] Username of recipient creator.`)
	// TODO: complex arg: ip_access_list
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `[Create:REQ,Update:OPT] Name of Recipient.`)
	createCmd.Flags().StringVar(&createReq.SharingCode, "sharing-code", "", `[Create:OPT,Update:IGN] The one-time sharing code provided by the data recipient.`)
	// TODO: array: tokens
	createCmd.Flags().Int64Var(&createReq.UpdatedAt, "updated-at", 0, `[Create:IGN,Update:IGN] Time at which the recipient was updated, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.UpdatedBy, "updated-by", "", `[Create:IGN,Update:IGN] Username of recipient updater.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share recipient.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Recipients.Create(ctx, createReq)
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

var deleteReq unitycatalog.DeleteRecipientRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", "", `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a share recipient.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	getCmd.Flags().StringVar(&getReq.Name, "name", "", `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a share recipient.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Recipients.Get(ctx, getReq)
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

var listReq unitycatalog.ListRecipientsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.DataRecipientGlobalMetastoreId, "data-recipient-global-metastore-id", "", `If not provided, all recipients will be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List share recipients.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Recipients.ListAll(ctx, listReq)
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

var rotateTokenReq unitycatalog.RotateRecipientToken

func init() {
	Cmd.AddCommand(rotateTokenCmd)
	// TODO: short flags

	rotateTokenCmd.Flags().Int64Var(&rotateTokenReq.ExistingTokenExpireInSeconds, "existing-token-expire-in-seconds", 0, `Required.`)
	rotateTokenCmd.Flags().StringVar(&rotateTokenReq.Name, "name", "", `Required.`)

}

var rotateTokenCmd = &cobra.Command{
	Use:   "rotate-token",
	Short: `Rotate a token.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Recipients.RotateToken(ctx, rotateTokenReq)
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
	Short: `Get share permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Recipients.SharePermissions(ctx, sharePermissionsReq)
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

var updateReq unitycatalog.UpdateRecipient

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Activated, "activated", false, `[Create:IGN,Update:IGN] A boolean status field showing whether the Recipient's activation URL has been exercised or not.`)
	updateCmd.Flags().StringVar(&updateReq.ActivationUrl, "activation-url", "", `[Create:IGN,Update:IGN] Full activation url to retrieve the access token.`)
	updateCmd.Flags().Var(&updateReq.AuthenticationType, "authentication-type", `[Create:REQ,Update:IGN] The delta sharing authentication type.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", "", `[Create:OPT,Update:OPT] Description about the recipient.`)
	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", 0, `[Create:IGN,Update:IGN] Time at which this recipient was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", "", `[Create:IGN,Update:IGN] Username of recipient creator.`)
	// TODO: complex arg: ip_access_list
	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `[Create:REQ,Update:OPT] Name of Recipient.`)
	updateCmd.Flags().StringVar(&updateReq.SharingCode, "sharing-code", "", `[Create:OPT,Update:IGN] The one-time sharing code provided by the data recipient.`)
	// TODO: array: tokens
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", 0, `[Create:IGN,Update:IGN] Time at which the recipient was updated, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", "", `[Create:IGN,Update:IGN] Username of recipient updater.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share recipient.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Recipients.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
