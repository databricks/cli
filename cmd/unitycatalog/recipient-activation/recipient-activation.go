package recipient_activation

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipient-activation",
	Short: `Databricks Delta Sharing: Recipient Activation REST API.`, // TODO: fix FirstSentence logic and append dot to summary
}

var getActivationUrlInfoReq unitycatalog.GetActivationUrlInfoRequest

func init() {
	Cmd.AddCommand(getActivationUrlInfoCmd)
	// TODO: short flags

	getActivationUrlInfoCmd.Flags().StringVar(&getActivationUrlInfoReq.ActivationUrl, "activation-url", "", `Required.`)

}

var getActivationUrlInfoCmd = &cobra.Command{
	Use:   "get-activation-url-info",
	Short: `Get a share activation URL Gets information about an Activation URL.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.RecipientActivation.GetActivationUrlInfo(ctx, getActivationUrlInfoReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var retrieveTokenReq unitycatalog.RetrieveTokenRequest

func init() {
	Cmd.AddCommand(retrieveTokenCmd)
	// TODO: short flags

	retrieveTokenCmd.Flags().StringVar(&retrieveTokenReq.ActivationUrl, "activation-url", "", `Required.`)

}

var retrieveTokenCmd = &cobra.Command{
	Use:   "retrieve-token",
	Short: `Get an access token RPC to retrieve access token with an activation token.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.RecipientActivation.RetrieveToken(ctx, retrieveTokenReq)
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
