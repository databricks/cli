// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package gitcredentials

import (
	"fmt"

	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/gitcredentials"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "git-credentials",
	Short: `Registers personal access token for Databricks to do operations on behalf of the user.`,
	Long: `Registers personal access token for Databricks to do operations on behalf of
  the user.
  
  See [more info].
  
  [more info]: https://docs.databricks.com/repos/get-access-tokens-from-git-provider.html`,
}

// start create command

var createReq gitcredentials.CreateCredentials

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.GitUsername, "git-username", createReq.GitUsername, `Git username.`)
	createCmd.Flags().StringVar(&createReq.PersonalAccessToken, "personal-access-token", createReq.PersonalAccessToken, `The personal access token used to authenticate to the corresponding Git provider.`)

}

var createCmd = &cobra.Command{
	Use:   "create GIT_PROVIDER",
	Short: `Create a credential entry.`,
	Long: `Create a credential entry.
  
  Creates a Git credential entry for the user. Only one Git credential per user
  is supported, so any attempts to create credentials if an entry already exists
  will fail. Use the PATCH endpoint to update existing credentials, or the
  DELETE endpoint to delete existing credentials.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		createReq.GitProvider = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GitCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq gitcredentials.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete CREDENTIAL_ID",
	Short: `Delete a credential.`,
	Long: `Delete a credential.
  
  Deletes the specified Git credential.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &deleteReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.GitCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq gitcredentials.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get CREDENTIAL_ID",
	Short: `Get a credential entry.`,
	Long: `Get a credential entry.
  
  Gets the Git credential with the specified credential ID.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &getReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GitCredentials.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get Git credentials.`,
	Long: `Get Git credentials.
  
  Lists the calling user's Git credentials. One credential per user is
  supported.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.GitCredentials.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq gitcredentials.UpdateCredentials

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.GitProvider, "git-provider", updateReq.GitProvider, `Git provider.`)
	updateCmd.Flags().StringVar(&updateReq.GitUsername, "git-username", updateReq.GitUsername, `Git username.`)
	updateCmd.Flags().StringVar(&updateReq.PersonalAccessToken, "personal-access-token", updateReq.PersonalAccessToken, `The personal access token used to authenticate to the corresponding Git provider.`)

}

var updateCmd = &cobra.Command{
	Use:   "update CREDENTIAL_ID",
	Short: `Update a credential.`,
	Long: `Update a credential.
  
  Updates the specified Git credential.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &updateReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.GitCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service GitCredentials

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
