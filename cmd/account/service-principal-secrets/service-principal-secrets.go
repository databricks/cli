// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principal_secrets

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
	Annotations: map[string]string{
		"package": "oauth2",
	},
}

// start create command

var createReq oauth2.CreateServicePrincipalSecretRequest
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var createCmd = &cobra.Command{
	Use:   "create SERVICE_PRINCIPAL_ID",
	Short: `Create service principal secret.`,
	Long: `Create service principal secret.
  
  Create a secret for the given service principal.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &createReq.ServicePrincipalId)
			if err != nil {
				return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
			}
		}

		response, err := a.ServicePrincipalSecrets.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command

var deleteReq oauth2.DeleteServicePrincipalSecretRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete SERVICE_PRINCIPAL_ID SECRET_ID",
	Short: `Delete service principal secret.`,
	Long: `Delete service principal secret.
  
  Delete a secret from the given service principal.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &deleteReq.ServicePrincipalId)
			if err != nil {
				return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
			}
			deleteReq.SecretId = args[1]
		}

		err = a.ServicePrincipalSecrets.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

var listReq oauth2.ListServicePrincipalSecretsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var listCmd = &cobra.Command{
	Use:   "list SERVICE_PRINCIPAL_ID",
	Short: `List service principal secrets.`,
	Long: `List service principal secrets.
  
  List all secrets associated with the given service principal. This operation
  only returns information about the secrets themselves and does not include the
  secret values.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &listReq.ServicePrincipalId)
			if err != nil {
				return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
			}
		}

		response, err := a.ServicePrincipalSecrets.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service ServicePrincipalSecrets
