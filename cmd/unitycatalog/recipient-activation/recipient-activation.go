package recipient_activation

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipient-activation",
	Short: `Databricks Delta Sharing: Recipient Activation REST API.`,
	Long:  `Databricks Delta Sharing: Recipient Activation REST API`,
}

// start get-activation-url-info command

var getActivationUrlInfoReq unitycatalog.GetActivationUrlInfoRequest

func init() {
	Cmd.AddCommand(getActivationUrlInfoCmd)
	// TODO: short flags

	getActivationUrlInfoCmd.Flags().StringVar(&getActivationUrlInfoReq.ActivationUrl, "activation-url", getActivationUrlInfoReq.ActivationUrl, `Required.`)

}

var getActivationUrlInfoCmd = &cobra.Command{
	Use:   "get-activation-url-info",
	Short: `Get a share activation URL.`,
	Long: `Get a share activation URL.
  
  Gets information about an Activation URL.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RecipientActivation.GetActivationUrlInfo(ctx, getActivationUrlInfoReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start retrieve-token command

var retrieveTokenReq unitycatalog.RetrieveTokenRequest

func init() {
	Cmd.AddCommand(retrieveTokenCmd)
	// TODO: short flags

	retrieveTokenCmd.Flags().StringVar(&retrieveTokenReq.ActivationUrl, "activation-url", retrieveTokenReq.ActivationUrl, `Required.`)

}

var retrieveTokenCmd = &cobra.Command{
	Use:   "retrieve-token",
	Short: `Get an access token.`,
	Long: `Get an access token.
  
  RPC to retrieve access token with an activation token. This is a public API
  without any authentication.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RecipientActivation.RetrieveToken(ctx, retrieveTokenReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service RecipientActivation
