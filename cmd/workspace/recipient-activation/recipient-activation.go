// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipient_activation

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "recipient-activation",
	Short: `Databricks Recipient Activation REST API.`,
	Long:  `Databricks Recipient Activation REST API`,
}

// start get-activation-url-info command

var getActivationUrlInfoReq sharing.GetActivationUrlInfoRequest
var getActivationUrlInfoJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getActivationUrlInfoCmd)
	// TODO: short flags
	getActivationUrlInfoCmd.Flags().Var(&getActivationUrlInfoJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getActivationUrlInfoCmd = &cobra.Command{
	Use:   "get-activation-url-info ACTIVATION_URL",
	Short: `Get a share activation URL.`,
	Long: `Get a share activation URL.
  
  Gets an activation URL for a share.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
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
			err = getActivationUrlInfoJson.Unmarshal(&getActivationUrlInfoReq)
			if err != nil {
				return err
			}
		} else {
			getActivationUrlInfoReq.ActivationUrl = args[0]
		}

		err = w.RecipientActivation.GetActivationUrlInfo(ctx, getActivationUrlInfoReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start retrieve-token command

var retrieveTokenReq sharing.RetrieveTokenRequest
var retrieveTokenJson flags.JsonFlag

func init() {
	Cmd.AddCommand(retrieveTokenCmd)
	// TODO: short flags
	retrieveTokenCmd.Flags().Var(&retrieveTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var retrieveTokenCmd = &cobra.Command{
	Use:   "retrieve-token ACTIVATION_URL",
	Short: `Get an access token.`,
	Long: `Get an access token.
  
  Retrieve access token with an activation url. This is a public API without any
  authentication.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
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
			err = retrieveTokenJson.Unmarshal(&retrieveTokenReq)
			if err != nil {
				return err
			}
		} else {
			retrieveTokenReq.ActivationUrl = args[0]
		}

		response, err := w.RecipientActivation.RetrieveToken(ctx, retrieveTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service RecipientActivation
