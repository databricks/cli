// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package secrets

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "secrets",
	Short: `The Secrets API allows you to manage secrets, secret scopes, and access permissions.`,
	Long: `The Secrets API allows you to manage secrets, secret scopes, and access
  permissions.
  
  Sometimes accessing data requires that you authenticate to external data
  sources through JDBC. Instead of directly entering your credentials into a
  notebook, use Databricks secrets to store your credentials and reference them
  in notebooks and jobs.
  
  Administrators, secret creators, and users granted permission can read
  Databricks secrets. While Databricks makes an effort to redact secret values
  that might be displayed in notebooks, it is not possible to prevent such users
  from reading secrets.`,
}

// start create-scope command

var createScopeReq workspace.CreateScope
var createScopeJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createScopeCmd)
	// TODO: short flags
	createScopeCmd.Flags().Var(&createScopeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createScopeCmd.Flags().StringVar(&createScopeReq.InitialManagePrincipal, "initial-manage-principal", createScopeReq.InitialManagePrincipal, `The principal that is initially granted MANAGE permission to the created scope.`)
	// TODO: complex arg: keyvault_metadata
	createScopeCmd.Flags().Var(&createScopeReq.ScopeBackendType, "scope-backend-type", `The backend type the scope will be created with.`)

}

var createScopeCmd = &cobra.Command{
	Use:   "create-scope",
	Short: `Create a new secret scope.`,
	Long: `Create a new secret scope.
  
  The scope name must consist of alphanumeric characters, dashes, underscores,
  and periods, and may not exceed 128 characters. The maximum number of scopes
  in a workspace is 100.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createScopeJson.Unmarshall(&createScopeReq)
		if err != nil {
			return err
		}
		createScopeReq.Scope = args[0]

		err = w.Secrets.CreateScope(ctx, createScopeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-acl command

var deleteAclReq workspace.DeleteAcl

func init() {
	Cmd.AddCommand(deleteAclCmd)
	// TODO: short flags

}

var deleteAclCmd = &cobra.Command{
	Use:   "delete-acl SCOPE PRINCIPAL",
	Short: `Delete an ACL.`,
	Long: `Delete an ACL.
  
  Deletes the given ACL on the given scope.
  
  Users must have the MANAGE permission to invoke this API. Throws
  RESOURCE_DOES_NOT_EXIST if no such secret scope, principal, or ACL exists.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		deleteAclReq.Scope = args[0]
		deleteAclReq.Principal = args[1]

		err = w.Secrets.DeleteAcl(ctx, deleteAclReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-scope command

var deleteScopeReq workspace.DeleteScope

func init() {
	Cmd.AddCommand(deleteScopeCmd)
	// TODO: short flags

}

var deleteScopeCmd = &cobra.Command{
	Use:   "delete-scope SCOPE",
	Short: `Delete a secret scope.`,
	Long: `Delete a secret scope.
  
  Deletes a secret scope.
  
  Throws RESOURCE_DOES_NOT_EXIST if the scope does not exist. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		deleteScopeReq.Scope = args[0]

		err = w.Secrets.DeleteScope(ctx, deleteScopeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-secret command

var deleteSecretReq workspace.DeleteSecret

func init() {
	Cmd.AddCommand(deleteSecretCmd)
	// TODO: short flags

}

var deleteSecretCmd = &cobra.Command{
	Use:   "delete-secret SCOPE KEY",
	Short: `Delete a secret.`,
	Long: `Delete a secret.
  
  Deletes the secret stored in this secret scope. You must have WRITE or
  MANAGE permission on the secret scope.
  
  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope or secret exists.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		deleteSecretReq.Scope = args[0]
		deleteSecretReq.Key = args[1]

		err = w.Secrets.DeleteSecret(ctx, deleteSecretReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get-acl command

var getAclReq workspace.GetAclRequest

func init() {
	Cmd.AddCommand(getAclCmd)
	// TODO: short flags

}

var getAclCmd = &cobra.Command{
	Use:   "get-acl SCOPE PRINCIPAL",
	Short: `Get secret ACL details.`,
	Long: `Get secret ACL details.
  
  Gets the details about the given ACL, such as the group and permission. Users
  must have the MANAGE permission to invoke this API.
  
  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getAclReq.Scope = args[0]
		getAclReq.Principal = args[1]

		response, err := w.Secrets.GetAcl(ctx, getAclReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-acls command

var listAclsReq workspace.ListAclsRequest

func init() {
	Cmd.AddCommand(listAclsCmd)
	// TODO: short flags

}

var listAclsCmd = &cobra.Command{
	Use:   "list-acls SCOPE",
	Short: `Lists ACLs.`,
	Long: `Lists ACLs.
  
  List the ACLs for a given secret scope. Users must have the MANAGE
  permission to invoke this API.
  
  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		listAclsReq.Scope = args[0]

		response, err := w.Secrets.ListAclsAll(ctx, listAclsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-scopes command

func init() {
	Cmd.AddCommand(listScopesCmd)

}

var listScopesCmd = &cobra.Command{
	Use:   "list-scopes",
	Short: `List all scopes.`,
	Long: `List all scopes.
  
  Lists all secret scopes available in the workspace.
  
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Secrets.ListScopesAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-secrets command

var listSecretsReq workspace.ListSecretsRequest

func init() {
	Cmd.AddCommand(listSecretsCmd)
	// TODO: short flags

}

var listSecretsCmd = &cobra.Command{
	Use:   "list-secrets SCOPE",
	Short: `List secret keys.`,
	Long: `List secret keys.
  
  Lists the secret keys that are stored at this scope. This is a metadata-only
  operation; secret data cannot be retrieved using this API. Users need the READ
  permission to make this call.
  
  The lastUpdatedTimestamp returned is in milliseconds since epoch. Throws
  RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		listSecretsReq.Scope = args[0]

		response, err := w.Secrets.ListSecretsAll(ctx, listSecretsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start put-acl command

var putAclReq workspace.PutAcl

func init() {
	Cmd.AddCommand(putAclCmd)
	// TODO: short flags

}

var putAclCmd = &cobra.Command{
	Use:   "put-acl SCOPE PRINCIPAL PERMISSION",
	Short: `Create/update an ACL.`,
	Long: `Create/update an ACL.
  
  Creates or overwrites the Access Control List (ACL) associated with the given
  principal (user or group) on the specified scope point.
  
  In general, a user or group will use the most powerful permission available to
  them, and permissions are ordered as follows:
  
  * MANAGE - Allowed to change ACLs, and read and write to this secret scope.
  * WRITE - Allowed to read and write to this secret scope. * READ - Allowed
  to read this secret scope and list what secrets are available.
  
  Note that in general, secret values can only be read from within a command on
  a cluster (for example, through a notebook). There is no API to read the
  actual secret value material outside of a cluster. However, the user's
  permission will be applied based on who is executing the command, and they
  must have at least READ permission.
  
  Users must have the MANAGE permission to invoke this API.
  
  The principal is a user or group name corresponding to an existing Databricks
  principal to be granted or revoked access.
  
  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  RESOURCE_ALREADY_EXISTS if a permission for the principal already exists.
  Throws INVALID_PARAMETER_VALUE if the permission is invalid. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		putAclReq.Scope = args[0]
		putAclReq.Principal = args[1]
		_, err = fmt.Sscan(args[2], &putAclReq.Permission)
		if err != nil {
			return fmt.Errorf("invalid PERMISSION: %s", args[2])
		}

		err = w.Secrets.PutAcl(ctx, putAclReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start put-secret command

var putSecretReq workspace.PutSecret

func init() {
	Cmd.AddCommand(putSecretCmd)
	// TODO: short flags

	putSecretCmd.Flags().StringVar(&putSecretReq.BytesValue, "bytes-value", putSecretReq.BytesValue, `If specified, value will be stored as bytes.`)
	putSecretCmd.Flags().StringVar(&putSecretReq.StringValue, "string-value", putSecretReq.StringValue, `If specified, note that the value will be stored in UTF-8 (MB4) form.`)

}

var putSecretCmd = &cobra.Command{
	Use:   "put-secret SCOPE KEY",
	Short: `Add a secret.`,
	Long: `Add a secret.
  
  Inserts a secret under the provided scope with the given name. If a secret
  already exists with the same name, this command overwrites the existing
  secret's value. The server encrypts the secret using the secret scope's
  encryption settings before storing it.
  
  You must have WRITE or MANAGE permission on the secret scope. The secret
  key must consist of alphanumeric characters, dashes, underscores, and periods,
  and cannot exceed 128 characters. The maximum allowed secret value size is 128
  KB. The maximum number of secrets in a given scope is 1000.
  
  The input fields "string_value" or "bytes_value" specify the type of the
  secret, which will determine the value returned when the secret value is
  requested. Exactly one must be specified.
  
  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  RESOURCE_LIMIT_EXCEEDED if maximum number of secrets in scope is exceeded.
  Throws INVALID_PARAMETER_VALUE if the key name or value length is invalid.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		putSecretReq.Scope = args[0]
		putSecretReq.Key = args[1]

		err = w.Secrets.PutSecret(ctx, putSecretReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Secrets
