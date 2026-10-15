// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package log_delivery

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	logdelivery "github.com/databricks/cli/libs/sdktypes/logdelivery/v1"
	sdklogdelivery "github.com/databricks/sdk-go/logdelivery/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log-delivery",
		Short: `These APIs manage log delivery configurations for this account.`,
		Long: `These APIs manage log delivery configurations for this account. The two
  supported log types for this API are _billable usage logs_ (AWS only) and
  _audit logs_ (AWS and GCP). This feature is in Public Preview. This feature
  works with all account ID types.

  Log delivery works with all account types. However, if your account is on the
  E2 version of the platform or on a select custom plan that allows multiple
  workspaces per account, you can optionally configure different storage
  destinations for each workspace. Log delivery status is also provided to know
  the latest status of log delivery attempts.

  The high-level flow of billable usage delivery (AWS only):

  1. **Create storage**: In AWS, [create a new AWS S3 bucket] with a specific
  bucket policy. Using Databricks APIs, call the Account API to create a
  [storage configuration object](:method:Storage/Create) that uses the bucket
  name.

  2. **Create credentials**: In AWS, create the appropriate AWS IAM role. For
  full details, including the required IAM role policies and trust relationship,
  see [Billable usage log delivery]. Using Databricks APIs, call the Account API
  to create a [credential configuration object](:method:Credentials/Create) that
  uses the IAM role's ARN.

  3. **Create log delivery configuration**: Using Databricks APIs, call the
  Account API to [create a log delivery
  configuration](:method:LogDelivery/Create) that uses the credential and
  storage configuration objects from previous steps. You can specify if the logs
  should include all events of that log type in your account (_Account level_
  delivery) or only events for a specific set of workspaces (_workspace level_
  delivery). Account level log delivery applies to all current and future
  workspaces plus account level logs, while workspace level log delivery solely
  delivers logs related to the specified workspaces. You can create multiple
  types of delivery configurations per account.

  For billable usage delivery (AWS only): * For more information about billable
  usage logs, see [Billable usage log delivery]. For the CSV schema, see the
  [Usage page]. * The delivery location is
  <bucket-name>/<prefix>/billable-usage/csv/, where <prefix> is the name of
  the optional delivery path prefix you set up during log delivery
  configuration. Files are named
  workspaceId=<workspace-id>-usageMonth=<month>.csv. * All billable usage logs
  apply to specific workspaces (_workspace level_ logs). You can aggregate usage
  for your entire account by creating an _account level_ delivery configuration
  that delivers logs for all current and future workspaces in your account. *
  The files are delivered daily by overwriting the month's CSV file for each
  workspace.

  For audit log delivery (AWS and GCP): * For more information about about audit
  log delivery, see Audit log delivery [AWS] or [GCP], which includes
  information about the used JSON schema. * The delivery location is
  <bucket-name>/<delivery-path-prefix>/workspaceId=<workspaceId>/date=<yyyy-mm-dd>/auditlogs_<internal-id>.json.
  Files may get overwritten with the same content multiple times to achieve
  exactly-once delivery. * If the audit log delivery configuration included
  specific workspace IDs, only _workspace-level_ audit logs for those workspaces
  are delivered. If the log delivery configuration applies to the entire account
  (_account level_ delivery configuration), the audit log delivery includes
  workspace-level audit logs for all workspaces in the account as well as
  account-level audit logs. See Audit log delivery [AWS] or [GCP] for details. *
  Auditable events are typically available in logs within 15 minutes.

  [AWS]: https://docs.databricks.com/administration-guide/account-settings/audit-logs.html
  [Billable usage log delivery]: https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html
  [GCP]: https://docs.databricks.com/gcp/en/admin/account-settings/audit-logs
  [Usage page]: https://docs.databricks.com/administration-guide/account-settings/usage.html
  [create a new AWS S3 bucket]: https://docs.databricks.com/administration-guide/account-api/aws-storage.html`,
		GroupID: "billing",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newPatchStatus())

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
	*logdelivery.CreateLogDeliveryConfigurationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq logdelivery.CreateLogDeliveryConfigurationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a new log delivery configuration.`
	cmd.Long = `Create a new log delivery configuration.

  Creates a new Databricks log delivery configuration to enable delivery of the
  specified type of logs to your storage location. This requires that you
  already created a [credential object](:method:Credentials/Create) (which
  encapsulates a cross-account service IAM role) and a [storage configuration
  object](:method:Storage/Create) (which encapsulates an S3 bucket).

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
  configuration](:method:LogDelivery/PatchStatus)).

  [Configure audit logging]: https://docs.databricks.com/administration-guide/account-settings/audit-logs.html
  [Deliver and access billable usage logs]: https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := logdelivery.CreateLogDeliveryConfigurationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.LogDelivery.CreateLogDeliveryConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := logdelivery.CreateLogDeliveryConfigurationResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdklogdelivery.GetLogDeliveryConfigurationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdklogdelivery.GetLogDeliveryConfigurationRequest

	cmd.Use = "get LOG_DELIVERY_CONFIGURATION_ID"
	cmd.Short = `Get log delivery configuration.`
	cmd.Long = `Get log delivery configuration.

  Gets a Databricks log delivery configuration object for an account, both
  specified by ID.

  Arguments:
    LOG_DELIVERY_CONFIGURATION_ID: The log delivery configuration id of customer`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No LOG_DELIVERY_CONFIGURATION_ID argument specified. Loading names for Log Delivery drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, a.LogDelivery.ListLogDeliveryConfigurationIter(ctx, sdklogdelivery.ListLogDeliveryConfigurationRequest{}), []string{"ConfigName"}, []string{"ConfigId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Log Delivery drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The log delivery configuration id of customer")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the log delivery configuration id of customer")
		}
		getReq.ConfigId = new(args[0])

		sdkReq := &getReq
		response, err := a.LogDelivery.GetLogDeliveryConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := logdelivery.GetLogDeliveryConfigurationResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*logdelivery.ListLogDeliveryConfigurationRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq logdelivery.ListLogDeliveryConfigurationRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.CredentialsId, "credentials-id", `The Credentials id to filter the search results with.`)
	flags.EnumVar(cmd.Flags(), &listReq.Status, "status", `The log delivery status to filter the search results with. Supported values: [DISABLED, ENABLED]`, "LogDeliveryConfigStatus", "DISABLED", "ENABLED")
	flags.StringPtrVar(cmd.Flags(), &listReq.StorageConfigurationId, "storage-configuration-id", `The Storage Configuration id to filter the search results with.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `Get all log delivery configurations.`
	cmd.Long = `Get all log delivery configurations.

  Gets all Databricks log delivery configurations associated with an account
  specified by ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := logdelivery.ListLogDeliveryConfigurationRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.LogDelivery.ListLogDeliveryConfigurationIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, logdelivery.LogDeliveryConfigurationFromSDK))
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

// start patch-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchStatusOverrides []func(
	*cobra.Command,
	*logdelivery.UpdateLogDeliveryConfigurationRequest,
)

func newPatchStatus() *cobra.Command {
	cmd := &cobra.Command{}
	var patchStatusReq logdelivery.UpdateLogDeliveryConfigurationRequest

	var patchStatusJson flags.JsonFlag

	cmd.Flags().Var(&patchStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-status LOG_DELIVERY_CONFIGURATION_ID STATUS"
	cmd.Short = `Enable or disable log delivery configuration.`
	cmd.Long = `Enable or disable log delivery configuration.

  Enables or disables a log delivery configuration. Deletion of delivery
  configurations is not supported, so disable log delivery configurations that
  are no longer needed. Note that you can't re-enable a delivery configuration
  if this would violate the delivery configuration limits described under
  [Create log delivery](:method:LogDelivery/Create).

  Arguments:
    LOG_DELIVERY_CONFIGURATION_ID: The log delivery configuration id of customer
    STATUS: Status of log delivery configuration. Set to ENABLED (enabled) or
      DISABLED (disabled). Defaults to ENABLED. You can [enable or disable
      the configuration](#operation/patch-log-delivery-config-status) later.
      Deletion of a configuration is not supported, so disable a log delivery
      configuration that is no longer needed.
      Supported values: [DISABLED, ENABLED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only LOG_DELIVERY_CONFIGURATION_ID as positional arguments. Provide 'status' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := patchStatusJson.Unmarshal(&patchStatusReq)
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
		patchStatusReq.ConfigId = new(args[0])
		if !cmd.Flags().Changed("json") {
			patchStatusReq.Status = logdelivery.LogDeliveryConfigStatus(args[1])
		}

		sdkReq, err := logdelivery.UpdateLogDeliveryConfigurationRequestToSDK(&patchStatusReq)
		if err != nil {
			return err
		}
		response, err := a.LogDelivery.UpdateLogDeliveryConfiguration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchStatusOverrides {
		fn(cmd, &patchStatusReq)
	}

	return cmd
}

// end service LogDelivery
