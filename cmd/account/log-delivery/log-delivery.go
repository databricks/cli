// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package log_delivery

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "log-delivery",
	Short: `These APIs manage log delivery configurations for this account.`,
	Long: `These APIs manage log delivery configurations for this account. The two
  supported log types for this API are _billable usage logs_ and _audit logs_.
  This feature is in Public Preview. This feature works with all account ID
  types.
  
  Log delivery works with all account types. However, if your account is on the
  E2 version of the platform or on a select custom plan that allows multiple
  workspaces per account, you can optionally configure different storage
  destinations for each workspace. Log delivery status is also provided to know
  the latest status of log delivery attempts. The high-level flow of billable
  usage delivery:
  
  1. **Create storage**: In AWS, [create a new AWS S3 bucket] with a specific
  bucket policy. Using Databricks APIs, call the Account API to create a
  [storage configuration object](#operation/create-storage-config) that uses the
  bucket name. 2. **Create credentials**: In AWS, create the appropriate AWS IAM
  role. For full details, including the required IAM role policies and trust
  relationship, see [Billable usage log delivery]. Using Databricks APIs, call
  the Account API to create a [credential configuration
  object](#operation/create-credential-config) that uses the IAM role's ARN. 3.
  **Create log delivery configuration**: Using Databricks APIs, call the Account
  API to [create a log delivery
  configuration](#operation/create-log-delivery-config) that uses the credential
  and storage configuration objects from previous steps. You can specify if the
  logs should include all events of that log type in your account (_Account
  level_ delivery) or only events for a specific set of workspaces (_workspace
  level_ delivery). Account level log delivery applies to all current and future
  workspaces plus account level logs, while workspace level log delivery solely
  delivers logs related to the specified workspaces. You can create multiple
  types of delivery configurations per account.
  
  For billable usage delivery: * For more information about billable usage logs,
  see [Billable usage log delivery]. For the CSV schema, see the [Usage page]. *
  The delivery location is <bucket-name>/<prefix>/billable-usage/csv/, where
  <prefix> is the name of the optional delivery path prefix you set up during
  log delivery configuration. Files are named
  workspaceId=<workspace-id>-usageMonth=<month>.csv. * All billable usage logs
  apply to specific workspaces (_workspace level_ logs). You can aggregate usage
  for your entire account by creating an _account level_ delivery configuration
  that delivers logs for all current and future workspaces in your account. *
  The files are delivered daily by overwriting the month's CSV file for each
  workspace.
  
  For audit log delivery: * For more information about about audit log delivery,
  see [Audit log delivery], which includes information about the used JSON
  schema. * The delivery location is
  <bucket-name>/<delivery-path-prefix>/workspaceId=<workspaceId>/date=<yyyy-mm-dd>/auditlogs_<internal-id>.json.
  Files may get overwritten with the same content multiple times to achieve
  exactly-once delivery. * If the audit log delivery configuration included
  specific workspace IDs, only _workspace-level_ audit logs for those workspaces
  are delivered. If the log delivery configuration applies to the entire account
  (_account level_ delivery configuration), the audit log delivery includes
  workspace-level audit logs for all workspaces in the account as well as
  account-level audit logs. See [Audit log delivery] for details. * Auditable
  events are typically available in logs within 15 minutes.
  
  [Audit log delivery]: https://docs.databricks.com/administration-guide/account-settings/audit-logs.html
  [Billable usage log delivery]: https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html
  [Usage page]: https://docs.databricks.com/administration-guide/account-settings/usage.html
  [create a new AWS S3 bucket]: https://docs.databricks.com/administration-guide/account-api/aws-storage.html`,
	Annotations: map[string]string{
		"package": "billing",
	},
}

// start create command

var createReq billing.WrappedCreateLogDeliveryConfiguration
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := a.LogDelivery.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command

var getReq billing.GetLogDeliveryRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get LOG_DELIVERY_CONFIGURATION_ID",
	Short: `Get log delivery configuration.`,
	Long: `Get log delivery configuration.
  
  Gets a Databricks log delivery configuration object for an account, both
  specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No LOG_DELIVERY_CONFIGURATION_ID argument specified. Loading names for Log Delivery drop-down."
				names, err := a.LogDelivery.LogDeliveryConfigurationConfigNameToConfigIdMap(ctx, billing.ListLogDeliveryRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Log Delivery drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Databricks log delivery configuration ID")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have databricks log delivery configuration id")
			}
			getReq.LogDeliveryConfigurationId = args[0]
		}

		response, err := a.LogDelivery.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

var listReq billing.ListLogDeliveryRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().StringVar(&listReq.CredentialsId, "credentials-id", listReq.CredentialsId, `Filter by credential configuration ID.`)
	listCmd.Flags().Var(&listReq.Status, "status", `Filter by status ENABLED or DISABLED.`)
	listCmd.Flags().StringVar(&listReq.StorageConfigurationId, "storage-configuration-id", listReq.StorageConfigurationId, `Filter by storage configuration ID.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all log delivery configurations.`,
	Long: `Get all log delivery configurations.
  
  Gets all Databricks log delivery configurations associated with an account
  specified by ID.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := a.LogDelivery.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start patch-status command

var patchStatusReq billing.UpdateLogDeliveryConfigurationStatusRequest
var patchStatusJson flags.JsonFlag

func init() {
	Cmd.AddCommand(patchStatusCmd)
	// TODO: short flags
	patchStatusCmd.Flags().Var(&patchStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var patchStatusCmd = &cobra.Command{
	Use:   "patch-status STATUS LOG_DELIVERY_CONFIGURATION_ID",
	Short: `Enable or disable log delivery configuration.`,
	Long: `Enable or disable log delivery configuration.
  
  Enables or disables a log delivery configuration. Deletion of delivery
  configurations is not supported, so disable log delivery configurations that
  are no longer needed. Note that you can't re-enable a delivery configuration
  if this would violate the delivery configuration limits described under
  [Create log delivery](#operation/create-log-delivery-config).`,

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
			err = patchStatusJson.Unmarshal(&patchStatusReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &patchStatusReq.Status)
			if err != nil {
				return fmt.Errorf("invalid STATUS: %s", args[0])
			}
			patchStatusReq.LogDeliveryConfigurationId = args[1]
		}

		err = a.LogDelivery.PatchStatus(ctx, patchStatusReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service LogDelivery
