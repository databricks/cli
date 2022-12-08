package git_credentials

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/gitcredentials"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "git-credentials",
	Short: `Registers personal access token for Databricks to do operations on behalf of the user.`,
}

var createReq gitcredentials.CreateCredentials

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.GitProvider, "git-provider", "", `Git provider.`)
	createCmd.Flags().StringVar(&createReq.GitUsername, "git-username", "", `Git username.`)
	createCmd.Flags().StringVar(&createReq.PersonalAccessToken, "personal-access-token", "", `The personal access token used to authenticate to the corresponding Git provider.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a credential entry.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.GitCredentials.Create(ctx, createReq)
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

var deleteReq gitcredentials.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.CredentialId, "credential-id", 0, `The ID for the corresponding credential to access.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a credential.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.GitCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq gitcredentials.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.CredentialId, "credential-id", 0, `The ID for the corresponding credential to access.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a credential entry.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.GitCredentials.Get(ctx, getReq)
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

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get Git credentials.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.GitCredentials.ListAll(ctx)
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

var updateReq gitcredentials.UpdateCredentials

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().Int64Var(&updateReq.CredentialId, "credential-id", 0, `The ID for the corresponding credential to access.`)
	updateCmd.Flags().StringVar(&updateReq.GitProvider, "git-provider", "", `Git provider.`)
	updateCmd.Flags().StringVar(&updateReq.GitUsername, "git-username", "", `Git username.`)
	updateCmd.Flags().StringVar(&updateReq.PersonalAccessToken, "personal-access-token", "", `The personal access token used to authenticate to the corresponding Git provider.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a credential.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.GitCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
