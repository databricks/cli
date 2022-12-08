package tokens

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/tokens"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tokens",
	Short: `The Token API allows you to create, list, and revoke tokens that can be used to authenticate and access Databricks REST APIs.`,
}

var createReq tokens.CreateTokenRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `Optional description to attach to the token.`)
	createCmd.Flags().Int64Var(&createReq.LifetimeSeconds, "lifetime-seconds", 0, `The lifetime of the token, in seconds.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a user token.`,
	Long: `Create a user token.
  
  Creates and returns a token for a user. If this call is made through token
  authentication, it creates a token with the same client ID as the
  authenticated token. If the user's token quota is exceeded, this call returns
  an error **QUOTA_EXCEEDED**.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tokens.Create(ctx, createReq)
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

var deleteReq tokens.RevokeTokenRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.TokenId, "token-id", "", `The ID of the token to be revoked.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Revoke token.`,
	Long: `Revoke token.
  
  Revokes an access token.
  
  If a token with the specified ID is not valid, this call returns an error
  **RESOURCE_DOES_NOT_EXIST**.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Tokens.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List tokens.`,
	Long: `List tokens.
  
  Lists all the valid tokens for a user-workspace pair.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tokens.ListAll(ctx)
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
