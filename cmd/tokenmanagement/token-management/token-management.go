package token_management

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/tokenmanagement"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "token-management",
	Short: `Enables administrators to get all tokens and delete tokens for other users.`, // TODO: fix FirstSentence logic and append dot to summary
}

var createOboTokenReq tokenmanagement.CreateOboTokenRequest

func init() {
	Cmd.AddCommand(createOboTokenCmd)
	// TODO: short flags

	createOboTokenCmd.Flags().StringVar(&createOboTokenReq.ApplicationId, "application-id", "", `Application ID of the service principal.`)
	createOboTokenCmd.Flags().StringVar(&createOboTokenReq.Comment, "comment", "", `Comment that describes the purpose of the token.`)
	// TODO: complex arg: lifetime_seconds

}

var createOboTokenCmd = &cobra.Command{
	Use:   "create-obo-token",
	Short: `Create on-behalf token Creates a token on behalf of a service principal.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.TokenManagement.CreateOboToken(ctx, createOboTokenReq)
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

var deleteTokenReq tokenmanagement.DeleteToken

func init() {
	Cmd.AddCommand(deleteTokenCmd)
	// TODO: short flags

	deleteTokenCmd.Flags().StringVar(&deleteTokenReq.TokenId, "token-id", "", `The ID of the token to get.`)

}

var deleteTokenCmd = &cobra.Command{
	Use:   "delete-token",
	Short: `Delete a token Deletes a token, specified by its ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.TokenManagement.DeleteToken(ctx, deleteTokenReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getTokenInfoReq tokenmanagement.GetTokenInfo

func init() {
	Cmd.AddCommand(getTokenInfoCmd)
	// TODO: short flags

	getTokenInfoCmd.Flags().StringVar(&getTokenInfoReq.TokenId, "token-id", "", `The ID of the token to get.`)

}

var getTokenInfoCmd = &cobra.Command{
	Use:   "get-token-info",
	Short: `Get token info Gets information about a token, specified by its ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.TokenManagement.GetTokenInfo(ctx, getTokenInfoReq)
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

var listTokensReq tokenmanagement.ListTokens

func init() {
	Cmd.AddCommand(listTokensCmd)
	// TODO: short flags

	listTokensCmd.Flags().StringVar(&listTokensReq.CreatedById, "created-by-id", "", `User ID of the user that created the token.`)
	listTokensCmd.Flags().StringVar(&listTokensReq.CreatedByUsername, "created-by-username", "", `Username of the user that created the token.`)

}

var listTokensCmd = &cobra.Command{
	Use:   "list-tokens",
	Short: `List all tokens Lists all tokens associated with the specified workspace or user.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.TokenManagement.ListTokensAll(ctx, listTokensReq)
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
