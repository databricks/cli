// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "secrets",
	Short:   `The Secrets API allows you to manage secrets, secret scopes, and access permissions.`,
	GroupID: "workspace",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `The Secrets API allows you to manage secrets, secret scopes, and access
  permissions.

  Sometimes accessing data requires that you authenticate to external data
  sources through JDBC. Instead of directly entering your credentials into a
  notebook, use Databricks secrets to store your credentials and reference them
  in notebooks and jobs.

  Administrators, secret creators, and users granted permission can read
  Databricks secrets. While Databricks makes an effort to redact secret values
  that might be displayed in notebooks, it is not possible to prevent such users
  from reading secrets.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "GA",
		"launch_stage_display": "GA",
	}

	// Add methods
	cmd.AddCommand(newCreateScope())
	cmd.AddCommand(newDeleteAcl())
	cmd.AddCommand(newDeleteScope())
	cmd.AddCommand(newDeleteSecret())
	cmd.AddCommand(newGetAcl())
	cmd.AddCommand(newGetSecret())
	cmd.AddCommand(newListAcls())
	cmd.AddCommand(newListScopes())
	cmd.AddCommand(newListSecrets())
	cmd.AddCommand(newPutAcl())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start create-scope command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createScopeOverrides []func(
	*cobra.Command,
	*workspace.CreateScope,
)

func newCreateScope() *cobra.Command {
	cmd := &cobra.Command{}

	var createScopeReq workspace.CreateScope
	var createScopeJson flags.JsonFlag

	cmd.Flags().Var(&createScopeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: backend_azure_keyvault
	cmd.Flags().StringVar(&createScopeReq.InitialManagePrincipal, "initial-manage-principal", createScopeReq.InitialManagePrincipal, `The principal that is initially granted MANAGE permission to the created scope.`)
	cmd.Flags().Var(&createScopeReq.ScopeBackendType, "scope-backend-type", `The backend type of the new scope. Supported values: [AZURE_KEYVAULT, DATABRICKS]`)

	cmd.Use = "create-scope SCOPE"
	cmd.Short = `Create a new secret scope.`
	cmd.Long = `Create a new secret scope.

  Creates a new secret scope.

  The scope name must consist of alphanumeric characters, dashes, underscores,
  and periods, and may not exceed 128 characters.

  If initial_manage_principal is specified, the initial ACL applied to the
  scope is applied to the supplied principal (user or group) with MANAGE
  permissions. The only supported principal for this option is the group
  users, which contains all users in the workspace. If
  initial_manage_principal is not specified, the initial ACL with MANAGE
  permission applied to the scope is assigned to the API request issuer's user
  identity.

  If scope_backend_type is AZURE_KEYVAULT, a secret scope is created
  with secrets from an existing Azure key vault. The caller must provide the key
  vault's resource ID and DNS name in backend_azure_keyvault. If
  scope_backend_type is DATABRICKS or is unspecified, an empty secret
  scope is created and stored in Databricks-managed storage.

  Throws RESOURCE_ALREADY_EXISTS if a scope with the given name already
  exists. Throws RESOURCE_LIMIT_EXCEEDED if maximum number of scopes in the
  workspace is exceeded. Throws INVALID_PARAMETER_VALUE if the scope name is
  invalid. Throws BAD_REQUEST if request violated constraints. Throws
  CUSTOMER_UNAUTHORIZED if the scope name is reserved for Databricks
  internal use. Throws UNAUTHENTICATED if Databricks cannot verify the
  caller's access to the Azure key vault.

  Arguments:
    SCOPE: Scope name requested by the user. Scope names are unique.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'scope' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createScopeJson.Unmarshal(&createScopeReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createScopeReq.Scope = args[0]
		}

		err = w.Secrets.CreateScope(ctx, createScopeReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createScopeOverrides {
		fn(cmd, &createScopeReq)
	}

	return cmd
}

// start delete-acl command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAclOverrides []func(
	*cobra.Command,
	*workspace.DeleteAcl,
)

func newDeleteAcl() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteAclReq workspace.DeleteAcl
	var deleteAclJson flags.JsonFlag

	cmd.Flags().Var(&deleteAclJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-acl SCOPE PRINCIPAL"
	cmd.Short = `Delete an ACL.`
	cmd.Long = `Delete an ACL.

  Deletes the given ACL on the given scope.

  Users must have the MANAGE permission to invoke this API.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope, principal, or ACL
  exists. Throws PERMISSION_DENIED if the user does not have permission to
  make this API call. Throws INVALID_PARAMETER_VALUE if the permission or
  principal is invalid.

  Arguments:
    SCOPE: The name of the scope to remove permissions from.
    PRINCIPAL: The principal to remove an existing ACL from.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'scope', 'principal' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteAclJson.Unmarshal(&deleteAclReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteAclReq.Scope = args[0]
		}
		if !cmd.Flags().Changed("json") {
			deleteAclReq.Principal = args[1]
		}

		err = w.Secrets.DeleteAcl(ctx, deleteAclReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteAclOverrides {
		fn(cmd, &deleteAclReq)
	}

	return cmd
}

// start delete-scope command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteScopeOverrides []func(
	*cobra.Command,
	*workspace.DeleteScope,
)

func newDeleteScope() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteScopeReq workspace.DeleteScope
	var deleteScopeJson flags.JsonFlag

	cmd.Flags().Var(&deleteScopeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-scope SCOPE"
	cmd.Short = `Delete a secret scope.`
	cmd.Long = `Delete a secret scope.

  Deletes a secret scope.

  Throws RESOURCE_DOES_NOT_EXIST if the scope does not exist. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.

  Arguments:
    SCOPE: Name of the scope to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'scope' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteScopeJson.Unmarshal(&deleteScopeReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteScopeReq.Scope = args[0]
		}

		err = w.Secrets.DeleteScope(ctx, deleteScopeReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteScopeOverrides {
		fn(cmd, &deleteScopeReq)
	}

	return cmd
}

// start delete-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSecretOverrides []func(
	*cobra.Command,
	*workspace.DeleteSecret,
)

func newDeleteSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSecretReq workspace.DeleteSecret
	var deleteSecretJson flags.JsonFlag

	cmd.Flags().Var(&deleteSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-secret SCOPE KEY"
	cmd.Short = `Delete a secret.`
	cmd.Long = `Delete a secret.

  Deletes the secret stored in this secret scope. You must have WRITE or
  MANAGE permission on the Secret Scope.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope or secret exists.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call. Throws BAD_REQUEST if the request is made against an Azure Key
  Vault-backed scope.

  Arguments:
    SCOPE: The name of the scope that contains the secret to delete.
    KEY: Name of the secret to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'scope', 'key' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteSecretJson.Unmarshal(&deleteSecretReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteSecretReq.Scope = args[0]
		}
		if !cmd.Flags().Changed("json") {
			deleteSecretReq.Key = args[1]
		}

		err = w.Secrets.DeleteSecret(ctx, deleteSecretReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSecretOverrides {
		fn(cmd, &deleteSecretReq)
	}

	return cmd
}

// start get-acl command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAclOverrides []func(
	*cobra.Command,
	*workspace.GetAclRequest,
)

func newGetAcl() *cobra.Command {
	cmd := &cobra.Command{}

	var getAclReq workspace.GetAclRequest

	cmd.Use = "get-acl SCOPE PRINCIPAL"
	cmd.Short = `Get secret ACL details.`
	cmd.Long = `Get secret ACL details.

  Describes the details about the given ACL, such as the group and permission.

  Users must have the MANAGE permission to invoke this API.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call. Throws INVALID_PARAMETER_VALUE if the permission or principal is
  invalid.

  Arguments:
    SCOPE: The name of the scope to fetch ACL information from.
    PRINCIPAL: The principal to fetch ACL information for.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getAclReq.Scope = args[0]
		getAclReq.Principal = args[1]

		response, err := w.Secrets.GetAcl(ctx, getAclReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAclOverrides {
		fn(cmd, &getAclReq)
	}

	return cmd
}

// start get-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSecretOverrides []func(
	*cobra.Command,
	*workspace.GetSecretRequest,
)

func newGetSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var getSecretReq workspace.GetSecretRequest

	cmd.Use = "get-secret SCOPE KEY"
	cmd.Short = `Get a secret.`
	cmd.Long = `Get a secret.

  Gets the bytes representation of a secret value for the specified scope and
  key. Users need the READ permission to make this call.

  The secret value is returned as bytes regardless of whether it was written
  with string_value or bytes_value, so the caller decides how to decode
  it. Values returned by this API are not redacted.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret or secret scope exists.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call. Azure Key Vault-backed scopes can also return the following errors:
  Throws INVALID_PARAMETER_VALUE if the secret name is not alphanumeric or
  is too long. Throws PERMISSION_DENIED if Databricks cannot access the key
  vault (HTTP 403). Throws MALFORMED_REQUEST if Databricks cannot access the
  key vault with any other 4xx error.

  Arguments:
    SCOPE: The name of the scope that contains the secret.
    KEY: Name of the secret to fetch value information.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSecretReq.Scope = args[0]
		getSecretReq.Key = args[1]

		response, err := w.Secrets.GetSecret(ctx, getSecretReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSecretOverrides {
		fn(cmd, &getSecretReq)
	}

	return cmd
}

// start list-acls command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listAclsOverrides []func(
	*cobra.Command,
	*workspace.ListAclsRequest,
)

func newListAcls() *cobra.Command {
	cmd := &cobra.Command{}

	var listAclsReq workspace.ListAclsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listAclsLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listAclsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list-acls SCOPE"
	cmd.Short = `Lists ACLs.`
	cmd.Long = `Lists ACLs.

  Lists the ACLs set on the given scope.

  Users must have the MANAGE permission to invoke this API.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.

  Arguments:
    SCOPE: The name of the scope to fetch ACL information from.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listAclsReq.Scope = args[0]

		response := w.Secrets.ListAcls(ctx, listAclsReq)
		if listAclsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listAclsLimit)
		}
		if listAclsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listAclsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listAclsOverrides {
		fn(cmd, &listAclsReq)
	}

	return cmd
}

// start list-scopes command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listScopesOverrides []func(
	*cobra.Command,
)

func newListScopes() *cobra.Command {
	cmd := &cobra.Command{}
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listScopesLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listScopesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list-scopes"
	cmd.Short = `List all scopes.`
	cmd.Long = `List all scopes.

  Lists all secret scopes available in the workspace.

  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response := w.Secrets.ListScopes(ctx)
		if listScopesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listScopesLimit)
		}
		if listScopesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listScopesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listScopesOverrides {
		fn(cmd)
	}

	return cmd
}

// start list-secrets command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSecretsOverrides []func(
	*cobra.Command,
	*workspace.ListSecretsRequest,
)

func newListSecrets() *cobra.Command {
	cmd := &cobra.Command{}

	var listSecretsReq workspace.ListSecretsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSecretsLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSecretsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list-secrets SCOPE"
	cmd.Short = `List secret keys.`
	cmd.Long = `List secret keys.

  Lists the secret keys that are stored at this scope. This is a metadata-only
  operation; secret data cannot be retrieved using this API. Users need the READ
  permission to make this call.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.

  Arguments:
    SCOPE: The name of the scope to list secrets within.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSecretsReq.Scope = args[0]

		response := w.Secrets.ListSecrets(ctx, listSecretsReq)
		if listSecretsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSecretsLimit)
		}
		if listSecretsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSecretsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSecretsOverrides {
		fn(cmd, &listSecretsReq)
	}

	return cmd
}

// start put-acl command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var putAclOverrides []func(
	*cobra.Command,
	*workspace.PutAcl,
)

func newPutAcl() *cobra.Command {
	cmd := &cobra.Command{}

	var putAclReq workspace.PutAcl
	var putAclJson flags.JsonFlag

	cmd.Flags().Var(&putAclJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "put-acl SCOPE PRINCIPAL PERMISSION"
	cmd.Short = `Create/update an ACL.`
	cmd.Long = `Create/update an ACL.

  Creates or overwrites the ACL associated with the given principal (user or
  group) on the specified scope point. In general, a user or group will use the
  most powerful permission available to them, and permissions are ordered as
  follows:

  * MANAGE - Allowed to change ACLs, and read and write to this secret
  scope. * WRITE - Allowed to read and write to this secret scope. *
  READ - Allowed to read this secret scope and list what secrets are
  available.

  Users must have the MANAGE permission to invoke this API.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  INVALID_PARAMETER_VALUE if the permission or principal is invalid. Throws
  PERMISSION_DENIED if the user does not have permission to make this API
  call.

  Arguments:
    SCOPE: The name of the scope to apply permissions to.
    PRINCIPAL: The user name or group name of an existing Databricks principal that the
      permission applies to.
    PERMISSION: The permission level applied to the principal.
      Supported values: [MANAGE, READ, WRITE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'scope', 'principal', 'permission' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := putAclJson.Unmarshal(&putAclReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			putAclReq.Scope = args[0]
		}
		if !cmd.Flags().Changed("json") {
			putAclReq.Principal = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &putAclReq.Permission)
			if err != nil {
				return fmt.Errorf("invalid PERMISSION: %s", args[2])
			}

		}

		err = w.Secrets.PutAcl(ctx, putAclReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range putAclOverrides {
		fn(cmd, &putAclReq)
	}

	return cmd
}

// end service Secrets
