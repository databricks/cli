// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principal_secrets

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-principal-secrets",
		Short: `These APIs enable administrators to manage service principal secrets.`,
		Long: `These APIs enable administrators to manage service principal secrets.
  
  You can use the generated secrets to obtain OAuth access tokens for a service
  principal, which can then be used to access Databricks Accounts and Workspace
  APIs. For more information, see [Authentication using OAuth tokens for service
  principals],
  
  In addition, the generated secrets can be used to configure the Databricks
  Terraform Provider to authenticate with the service principal. For more
  information, see [Databricks Terraform Provider].
  
  [Authentication using OAuth tokens for service principals]: https://docs.databricks.com/dev-tools/authentication-oauth.html
  [Databricks Terraform Provider]: https://github.com/databricks/terraform-provider-databricks/blob/master/docs/index.md#authenticating-with-service-principal`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*oauth2.CreateServicePrincipalSecretRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq oauth2.CreateServicePrincipalSecretRequest

	// TODO: short flags

	cmd.Use = "create SERVICE_PRINCIPAL_ID"
	cmd.Short = `Create service principal secret.`
	cmd.Long = `Create service principal secret.
  
  Create a secret for the given service principal.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &createReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}

		response, err := a.ServicePrincipalSecrets.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*oauth2.DeleteServicePrincipalSecretRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq oauth2.DeleteServicePrincipalSecretRequest

	// TODO: short flags

	cmd.Use = "delete SERVICE_PRINCIPAL_ID SECRET_ID"
	cmd.Short = `Delete service principal secret.`
	cmd.Long = `Delete service principal secret.
  
  Delete a secret from the given service principal.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal ID.
    SECRET_ID: The secret ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}
		deleteReq.SecretId = args[1]

		err = a.ServicePrincipalSecrets.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*oauth2.ListServicePrincipalSecretsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq oauth2.ListServicePrincipalSecretsRequest

	// TODO: short flags

	cmd.Use = "list SERVICE_PRINCIPAL_ID"
	cmd.Short = `List service principal secrets.`
	cmd.Long = `List service principal secrets.
  
  List all secrets associated with the given service principal. This operation
  only returns information about the secrets themselves and does not include the
  secret values.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &listReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}

		response := a.ServicePrincipalSecrets.List(ctx, listReq)

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// end service ServicePrincipalSecrets
