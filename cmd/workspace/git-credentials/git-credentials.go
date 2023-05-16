// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package git_credentials

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
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

var createReq workspace.CreateCredentials

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GitCredentials.CredentialInfoGitProviderToCredentialIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Git provider")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have git provider")
		}
		createReq.GitProvider = args[0]

		response, err := w.GitCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq workspace.DeleteGitCredentialRequest

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GitCredentials.CredentialInfoGitProviderToCredentialIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The ID for the corresponding credential to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding credential to access")
		}
		_, err = fmt.Sscan(args[0], &deleteReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		err = w.GitCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq workspace.GetGitCredentialRequest

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GitCredentials.CredentialInfoGitProviderToCredentialIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The ID for the corresponding credential to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding credential to access")
		}
		_, err = fmt.Sscan(args[0], &getReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		response, err := w.GitCredentials.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.GitCredentials.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq workspace.UpdateCredentials

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.GitCredentials.CredentialInfoGitProviderToCredentialIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The ID for the corresponding credential to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding credential to access")
		}
		_, err = fmt.Sscan(args[0], &updateReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		err = w.GitCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service GitCredentials
