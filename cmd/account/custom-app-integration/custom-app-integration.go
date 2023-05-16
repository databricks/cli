// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package custom_app_integration

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "custom-app-integration",
	Short: `These APIs enable administrators to manage custom oauth app integrations, which is required for adding/using Custom OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.`,
	Long: `These APIs enable administrators to manage custom oauth app integrations,
  which is required for adding/using Custom OAuth App Integration like Tableau
  Cloud for Databricks in AWS cloud.
  
  **Note:** You can only add/use the OAuth custom application integrations when
  OAuth enrollment status is enabled. For more details see
  :method:OAuthEnrollment/create`,
}

// start create command

var createReq oauth2.CreateCustomAppIntegration
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.Confidential, "confidential", createReq.Confidential, `indicates if an oauth client-secret should be generated.`)
	// TODO: complex arg: token_access_policy

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create Custom OAuth App Integration.`,
	Long: `Create Custom OAuth App Integration.
  
  Create Custom OAuth App Integration.
  
  You can retrieve the custom oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshal(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &createReq.RedirectUrls)
		if err != nil {
			return fmt.Errorf("invalid REDIRECT_URLS: %s", args[1])
		}

		response, err := a.CustomAppIntegration.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq oauth2.DeleteCustomAppIntegrationRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete INTEGRATION_ID",
	Short: `Delete Custom OAuth App Integration.`,
	Long: `Delete Custom OAuth App Integration.
  
  Delete an existing Custom OAuth App Integration. You can retrieve the custom
  oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		deleteReq.IntegrationId = args[0]

		err = a.CustomAppIntegration.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq oauth2.GetCustomAppIntegrationRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get INTEGRATION_ID",
	Short: `Get OAuth Custom App Integration.`,
	Long: `Get OAuth Custom App Integration.
  
  Gets the Custom OAuth App Integration for the given integration id.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		getReq.IntegrationId = args[0]

		response, err := a.CustomAppIntegration.Get(ctx, getReq)
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
	Short: `Get custom oauth app integrations.`,
	Long: `Get custom oauth app integrations.
  
  Get the list of custom oauth app integrations for the specified Databricks
  Account`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.CustomAppIntegration.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq oauth2.UpdateCustomAppIntegration
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: redirect_urls
	// TODO: complex arg: token_access_policy

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Updates Custom OAuth App Integration.`,
	Long: `Updates Custom OAuth App Integration.
  
  Updates an existing custom OAuth App Integration. You can retrieve the custom
  oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = updateJson.Unmarshal(&updateReq)
		if err != nil {
			return err
		}
		updateReq.IntegrationId = args[0]

		err = a.CustomAppIntegration.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service CustomAppIntegration
