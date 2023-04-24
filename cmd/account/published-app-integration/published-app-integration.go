// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package published_app_integration

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "published-app-integration",
	Short: `These APIs enable administrators to manage published oauth app integrations, which is required for adding/using Published OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.`,
	Long: `These APIs enable administrators to manage published oauth app integrations,
  which is required for adding/using Published OAuth App Integration like
  Tableau Cloud for Databricks in AWS cloud.
  
  **Note:** You can only add/use the OAuth published application integrations
  when OAuth enrollment status is enabled. For more details see
  :method:OAuthEnrollment/create`,
}

// start create command

var createReq oauth2.CreatePublishedAppIntegration
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.AppId, "app-id", createReq.AppId, `app_id of the oauth published app integration.`)
	// TODO: complex arg: token_access_policy

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create Published OAuth App Integration.`,
	Long: `Create Published OAuth App Integration.
  
  Create Published OAuth App Integration.
  
  You can retrieve the published oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}

		response, err := a.PublishedAppIntegration.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq oauth2.DeletePublishedAppIntegrationRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete INTEGRATION_ID",
	Short: `Delete Published OAuth App Integration.`,
	Long: `Delete Published OAuth App Integration.
  
  Delete an existing Published OAuth App Integration. You can retrieve the
  published oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		deleteReq.IntegrationId = args[0]

		err = a.PublishedAppIntegration.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq oauth2.GetPublishedAppIntegrationRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get INTEGRATION_ID",
	Short: `Get OAuth Published App Integration.`,
	Long: `Get OAuth Published App Integration.
  
  Gets the Published OAuth App Integration for the given integration id.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		getReq.IntegrationId = args[0]

		response, err := a.PublishedAppIntegration.Get(ctx, getReq)
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
	Short: `Get published oauth app integrations.`,
	Long: `Get published oauth app integrations.
  
  Get the list of published oauth app integrations for the specified Databricks
  Account`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.PublishedAppIntegration.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq oauth2.UpdatePublishedAppIntegration
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: token_access_policy

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Updates Published OAuth App Integration.`,
	Long: `Updates Published OAuth App Integration.
  
  Updates an existing published OAuth App Integration. You can retrieve the
  published oauth app integration via :method:get.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.IntegrationId = args[0]

		err = a.PublishedAppIntegration.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service PublishedAppIntegration
