package log_delivery

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "log-delivery",
	Short: `These APIs manage log delivery configurations for this account.`,
}

var createReq billing.WrappedCreateLogDeliveryConfiguration

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: log_delivery_configuration

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new log delivery configuration.`,
	Long: `Create a new log delivery configuration.
  
  Creates a new Databricks log delivery configuration to enable delivery of the
  specified type of logs to your storage location. This requires that you
  already created a [credential object](#operation/create-credential-config)
  (which encapsulates a cross-account service IAM role) and a [storage
  configuration object](#operation/create-storage-config) (which encapsulates an
  S3 bucket).
  
  For full details, including the required IAM role policies and bucket
  policies, see [Deliver and access billable usage logs] or [Configure audit
  logging].
  
  **Note**: There is a limit on the number of log delivery configurations
  available per account (each limit applies separately to each log type
  including billable usage and audit logs). You can create a maximum of two
  enabled account-level delivery configurations (configurations without a
  workspace filter) per type. Additionally, you can create two enabled
  workspace-level delivery configurations per workspace for each log type, which
  means that the same workspace ID can occur in the workspace filter for no more
  than two delivery configurations per log type.
  
  You cannot delete a log delivery configuration, but you can disable it (see
  [Enable or disable log delivery
  configuration](#operation/patch-log-delivery-config-status)).
  
  [Configure audit logging]: https://docs.databricks.com/administration-guide/account-settings/audit-logs.html
  [Deliver and access billable usage logs]: https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.LogDelivery.Create(ctx, createReq)
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

var getReq billing.GetLogDeliveryRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.LogDeliveryConfigurationId, "log-delivery-configuration-id", "", `Databricks log delivery configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get log delivery configuration.`,
	Long: `Get log delivery configuration.
  
  Gets a Databricks log delivery configuration object for an account, both
  specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.LogDelivery.Get(ctx, getReq)
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

var listReq billing.ListLogDeliveryRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CredentialsId, "credentials-id", "", `Filter by credential configuration ID.`)
	listCmd.Flags().Var(&listReq.Status, "status", `Filter by status ENABLED or DISABLED.`)
	listCmd.Flags().StringVar(&listReq.StorageConfigurationId, "storage-configuration-id", "", `Filter by storage configuration ID.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all log delivery configurations.`,
	Long: `Get all log delivery configurations.
  
  Gets all Databricks log delivery configurations associated with an account
  specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.LogDelivery.ListAll(ctx, listReq)
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

var patchStatusReq billing.UpdateLogDeliveryConfigurationStatusRequest

func init() {
	Cmd.AddCommand(patchStatusCmd)
	// TODO: short flags

	patchStatusCmd.Flags().StringVar(&patchStatusReq.LogDeliveryConfigurationId, "log-delivery-configuration-id", "", `Databricks log delivery configuration ID.`)
	patchStatusCmd.Flags().Var(&patchStatusReq.Status, "status", `Status of log delivery configuration.`)

}

var patchStatusCmd = &cobra.Command{
	Use:   "patch-status",
	Short: `Enable or disable log delivery configuration.`,
	Long: `Enable or disable log delivery configuration.
  
  Enables or disables a log delivery configuration. Deletion of delivery
  configurations is not supported, so disable log delivery configurations that
  are no longer needed. Note that you can't re-enable a delivery configuration
  if this would violate the delivery configuration limits described under
  [Create log delivery](#operation/create-log-delivery-config).`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.LogDelivery.PatchStatus(ctx, patchStatusReq)
		if err != nil {
			return err
		}

		return nil
	},
}
