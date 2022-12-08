package secrets

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/secrets"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "secrets",
	Short: `The Secrets API allows you to manage secrets, secret scopes, and access permissions.`,
}

var createScopeReq secrets.CreateScope

func init() {
	Cmd.AddCommand(createScopeCmd)
	// TODO: short flags

	createScopeCmd.Flags().StringVar(&createScopeReq.InitialManagePrincipal, "initial-manage-principal", "", `The principal that is initially granted MANAGE permission to the created scope.`)
	// TODO: complex arg: keyvault_metadata
	createScopeCmd.Flags().StringVar(&createScopeReq.Scope, "scope", "", `Scope name requested by the user.`)
	// TODO: complex arg: scope_backend_type

}

var createScopeCmd = &cobra.Command{
	Use:   "create-scope",
	Short: `Create a new secret scope.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.CreateScope(ctx, createScopeReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var deleteAclReq secrets.DeleteAcl

func init() {
	Cmd.AddCommand(deleteAclCmd)
	// TODO: short flags

	deleteAclCmd.Flags().StringVar(&deleteAclReq.Principal, "principal", "", `The principal to remove an existing ACL from.`)
	deleteAclCmd.Flags().StringVar(&deleteAclReq.Scope, "scope", "", `The name of the scope to remove permissions from.`)

}

var deleteAclCmd = &cobra.Command{
	Use:   "delete-acl",
	Short: `Delete an ACL.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.DeleteAcl(ctx, deleteAclReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var deleteScopeReq secrets.DeleteScope

func init() {
	Cmd.AddCommand(deleteScopeCmd)
	// TODO: short flags

	deleteScopeCmd.Flags().StringVar(&deleteScopeReq.Scope, "scope", "", `Name of the scope to delete.`)

}

var deleteScopeCmd = &cobra.Command{
	Use:   "delete-scope",
	Short: `Delete a secret scope.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.DeleteScope(ctx, deleteScopeReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var deleteSecretReq secrets.DeleteSecret

func init() {
	Cmd.AddCommand(deleteSecretCmd)
	// TODO: short flags

	deleteSecretCmd.Flags().StringVar(&deleteSecretReq.Key, "key", "", `Name of the secret to delete.`)
	deleteSecretCmd.Flags().StringVar(&deleteSecretReq.Scope, "scope", "", `The name of the scope that contains the secret to delete.`)

}

var deleteSecretCmd = &cobra.Command{
	Use:   "delete-secret",
	Short: `Delete a secret.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.DeleteSecret(ctx, deleteSecretReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getAclReq secrets.GetAcl

func init() {
	Cmd.AddCommand(getAclCmd)
	// TODO: short flags

	getAclCmd.Flags().StringVar(&getAclReq.Principal, "principal", "", `The principal to fetch ACL information for.`)
	getAclCmd.Flags().StringVar(&getAclReq.Scope, "scope", "", `The name of the scope to fetch ACL information from.`)

}

var getAclCmd = &cobra.Command{
	Use:   "get-acl",
	Short: `Get secret ACL details.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Secrets.GetAcl(ctx, getAclReq)
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

var listAclsReq secrets.ListAcls

func init() {
	Cmd.AddCommand(listAclsCmd)
	// TODO: short flags

	listAclsCmd.Flags().StringVar(&listAclsReq.Scope, "scope", "", `The name of the scope to fetch ACL information from.`)

}

var listAclsCmd = &cobra.Command{
	Use:   "list-acls",
	Short: `Lists ACLs.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Secrets.ListAclsAll(ctx, listAclsReq)
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
	Cmd.AddCommand(listScopesCmd)

}

var listScopesCmd = &cobra.Command{
	Use:   "list-scopes",
	Short: `List all scopes.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Secrets.ListScopesAll(ctx)
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

var listSecretsReq secrets.ListSecrets

func init() {
	Cmd.AddCommand(listSecretsCmd)
	// TODO: short flags

	listSecretsCmd.Flags().StringVar(&listSecretsReq.Scope, "scope", "", `The name of the scope to list secrets within.`)

}

var listSecretsCmd = &cobra.Command{
	Use:   "list-secrets",
	Short: `List secret keys.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Secrets.ListSecretsAll(ctx, listSecretsReq)
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

var putAclReq secrets.PutAcl

func init() {
	Cmd.AddCommand(putAclCmd)
	// TODO: short flags

	// TODO: complex arg: permission
	putAclCmd.Flags().StringVar(&putAclReq.Principal, "principal", "", `The principal in which the permission is applied.`)
	putAclCmd.Flags().StringVar(&putAclReq.Scope, "scope", "", `The name of the scope to apply permissions to.`)

}

var putAclCmd = &cobra.Command{
	Use:   "put-acl",
	Short: `Create/update an ACL.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.PutAcl(ctx, putAclReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var putSecretReq secrets.PutSecret

func init() {
	Cmd.AddCommand(putSecretCmd)
	// TODO: short flags

	putSecretCmd.Flags().StringVar(&putSecretReq.BytesValue, "bytes-value", "", `If specified, value will be stored as bytes.`)
	putSecretCmd.Flags().StringVar(&putSecretReq.Key, "key", "", `A unique name to identify the secret.`)
	putSecretCmd.Flags().StringVar(&putSecretReq.Scope, "scope", "", `The name of the scope to which the secret will be associated with.`)
	putSecretCmd.Flags().StringVar(&putSecretReq.StringValue, "string-value", "", `If specified, note that the value will be stored in UTF-8 (MB4) form.`)

}

var putSecretCmd = &cobra.Command{
	Use:   "put-secret",
	Short: `Add a secret.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Secrets.PutSecret(ctx, putSecretReq)
		if err != nil {
			return err
		}

		return nil
	},
}
