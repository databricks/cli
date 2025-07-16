// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principal_secrets_proxy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-principal-secrets-proxy",
		Short: `These APIs enable administrators to manage service principal secrets at the workspace level.`,
		Long: `These APIs enable administrators to manage service principal secrets at the
  workspace level. To use these APIs, the service principal must be first added
  to the current workspace.
  
  You can use the generated secrets to obtain OAuth access tokens for a service
  principal, which can then be used to access Databricks Accounts and Workspace
  APIs. For more information, see [Authentication using OAuth tokens for service
  principals].
  
  In addition, the generated secrets can be used to configure the Databricks
  Terraform Providerto authenticate with the service principal. For more
  information, see [Databricks Terraform Provider].
  
  [Authentication using OAuth tokens for service principals]: https://docs.databricks.com/dev-tools/authentication-oauth.html
  [Databricks Terraform Provider]: https://github.com/databricks/terraform-provider-databricks/blob/master/docs/index.md#authenticating-with-service-principal`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newList())

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
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Lifetime, "lifetime", createReq.Lifetime, `The lifetime of the secret in seconds.`)

	cmd.Use = "create SERVICE_PRINCIPAL_ID"
	cmd.Short = `Create service principal secret.`
	cmd.Long = `Create service principal secret.
  
  Create a secret for the given service principal.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createReq.ServicePrincipalId = args[0]

		response, err := w.ServicePrincipalSecretsProxy.Create(ctx, createReq)
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

	cmd.Use = "delete SERVICE_PRINCIPAL_ID SECRET_ID"
	cmd.Short = `Delete service principal secret.`
	cmd.Long = `Delete service principal secret.
  
  Delete a secret from the given service principal.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal ID.
    SECRET_ID: The secret ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.ServicePrincipalId = args[0]
		deleteReq.SecretId = args[1]

		err = w.ServicePrincipalSecretsProxy.Delete(ctx, deleteReq)
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

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `An opaque page token which was the next_page_token in the response of the previous request to list the secrets for this service principal.`)

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
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.ServicePrincipalId = args[0]

		response := w.ServicePrincipalSecretsProxy.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// end service ServicePrincipalSecretsProxy
