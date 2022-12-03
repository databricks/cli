package registry_webhooks

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "registry-webhooks",
}

var createReq mlflow.CreateRegistryWebhook

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Description, "description", "", `User-specified description for the webhook.`)
	// TODO: complex arg: events
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	createCmd.Flags().StringVar(&createReq.ModelName, "model-name", "", `Name of the model whose events would trigger this webhook.`)
	// TODO: complex arg: status

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a webhook **NOTE**: This endpoint is in Public Preview.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.RegistryWebhooks.Create(ctx, createReq)
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

var deleteReq mlflow.DeleteRegistryWebhookRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Webhook ID required to delete a registry webhook.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a webhook **NOTE:** This endpoint is in Public Preview.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.RegistryWebhooks.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var listReq mlflow.ListRegistryWebhooksRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	// TODO: complex arg: events
	listCmd.Flags().StringVar(&listReq.ModelName, "model-name", "", `If not specified, all webhooks associated with the specified events are listed, regardless of their associated model.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", "", `Token indicating the page of artifact results to fetch.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List registry webhooks **NOTE:** This endpoint is in Public Preview.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.RegistryWebhooks.ListAll(ctx, listReq)
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

var testReq mlflow.TestRegistryWebhookRequest

func init() {
	Cmd.AddCommand(testCmd)
	// TODO: short flags

	// TODO: complex arg: event
	testCmd.Flags().StringVar(&testReq.Id, "id", "", `Webhook ID.`)

}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: `Test a webhook **NOTE:** This endpoint is in Public Preview.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.RegistryWebhooks.Test(ctx, testReq)
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

var updateReq mlflow.UpdateRegistryWebhook

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Description, "description", "", `User-specified description for the webhook.`)
	// TODO: complex arg: events
	// TODO: complex arg: http_url_spec
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Webhook ID.`)
	// TODO: complex arg: job_spec
	// TODO: complex arg: status

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a webhook **NOTE:** This endpoint is in Public Preview.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.RegistryWebhooks.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
