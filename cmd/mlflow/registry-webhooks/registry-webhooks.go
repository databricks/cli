// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package registry_webhooks

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "registry-webhooks",
}

// start create command

var createReq mlflow.CreateRegistryWebhook
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `User-specified description for the webhook.`)
	// TODO: array: events
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	createCmd.Flags().StringVar(&createReq.ModelName, "model-name", createReq.ModelName, `Name of the model whose events would trigger this webhook.`)
	createCmd.Flags().Var(&createReq.Status, "status", `This describes an enum.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a webhook.`,
	Long: `Create a webhook.
  
  **NOTE**: This endpoint is in Public Preview.
  
  Creates a registry webhook.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegistryWebhooks.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteRegistryWebhookRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", deleteReq.Id, `Webhook ID required to delete a registry webhook.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a webhook.`,
	Long: `Delete a webhook.
  
  **NOTE:** This endpoint is in Public Preview.
  
  Deletes a registry webhook.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegistryWebhooks.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start list command

var listReq mlflow.ListRegistryWebhooksRequest
var listJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: events
	listCmd.Flags().StringVar(&listReq.ModelName, "model-name", listReq.ModelName, `If not specified, all webhooks associated with the specified events are listed, regardless of their associated model.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Token indicating the page of artifact results to fetch.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List registry webhooks.`,
	Long: `List registry webhooks.
  
  **NOTE:** This endpoint is in Public Preview.
  
  Lists all registry webhooks.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = listJson.Unmarshall(&listReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegistryWebhooks.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start test command

var testReq mlflow.TestRegistryWebhookRequest
var testJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(testCmd)
	// TODO: short flags
	testCmd.Flags().Var(&testJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	testCmd.Flags().Var(&testReq.Event, "event", `If event is specified, the test trigger uses the specified event.`)
	testCmd.Flags().StringVar(&testReq.Id, "id", testReq.Id, `Webhook ID.`)

}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: `Test a webhook.`,
	Long: `Test a webhook.
  
  **NOTE:** This endpoint is in Public Preview.
  
  Tests a registry webhook.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = testJson.Unmarshall(&testReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegistryWebhooks.Test(ctx, testReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq mlflow.UpdateRegistryWebhook
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `User-specified description for the webhook.`)
	// TODO: array: events
	// TODO: complex arg: http_url_spec
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Webhook ID.`)
	// TODO: complex arg: job_spec
	updateCmd.Flags().Var(&updateReq.Status, "status", `This describes an enum.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a webhook.`,
	Long: `Update a webhook.
  
  **NOTE:** This endpoint is in Public Preview.
  
  Updates a registry webhook.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegistryWebhooks.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service RegistryWebhooks
