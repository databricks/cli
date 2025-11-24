// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspaces

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: `These APIs manage workspaces for this account.`,
		Long: `These APIs manage workspaces for this account. A Databricks workspace is an
  environment for accessing all of your Databricks assets. The workspace
  organizes objects (notebooks, libraries, and experiments) into folders, and
  provides access to data and computational resources such as clusters and jobs.

  These endpoints are available if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,
		GroupID: "provisioning",
		Annotations: map[string]string{
			"package": "provisioning",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

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
	*provisioning.CreateWorkspaceRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq provisioning.CreateWorkspaceRequest
	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.AwsRegion, "aws-region", createReq.AwsRegion, ``)
	cmd.Flags().StringVar(&createReq.Cloud, "cloud", createReq.Cloud, `The cloud name.`)
	// TODO: complex arg: cloud_resource_container
	cmd.Flags().Var(&createReq.ComputeMode, "compute-mode", `If the compute mode is SERVERLESS, a serverless workspace is created that comes pre-configured with serverless compute and default storage, providing a fully-managed, enterprise-ready SaaS experience. Supported values: [HYBRID, SERVERLESS]`)
	cmd.Flags().StringVar(&createReq.CredentialsId, "credentials-id", createReq.CredentialsId, `ID of the workspace's credential configuration object.`)
	// TODO: map via StringToStringVar: custom_tags
	cmd.Flags().StringVar(&createReq.DeploymentName, "deployment-name", createReq.DeploymentName, `The deployment name defines part of the subdomain for the workspace.`)
	// TODO: complex arg: gcp_managed_network_config
	// TODO: complex arg: gke_config
	cmd.Flags().StringVar(&createReq.Location, "location", createReq.Location, `The Google Cloud region of the workspace data plane in your Google account (for example, us-east4).`)
	cmd.Flags().StringVar(&createReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", createReq.ManagedServicesCustomerManagedKeyId, `The ID of the workspace's managed services encryption key configuration object.`)
	cmd.Flags().StringVar(&createReq.NetworkConnectivityConfigId, "network-connectivity-config-id", createReq.NetworkConnectivityConfigId, `The object ID of network connectivity config.`)
	cmd.Flags().StringVar(&createReq.NetworkId, "network-id", createReq.NetworkId, `The ID of the workspace's network configuration object.`)
	cmd.Flags().Var(&createReq.PricingTier, "pricing-tier", `Supported values: [
  COMMUNITY_EDITION,
  DEDICATED,
  ENTERPRISE,
  PREMIUM,
  STANDARD,
  UNKNOWN,
]`)
	cmd.Flags().StringVar(&createReq.PrivateAccessSettingsId, "private-access-settings-id", createReq.PrivateAccessSettingsId, `ID of the workspace's private access settings object.`)
	cmd.Flags().StringVar(&createReq.StorageConfigurationId, "storage-configuration-id", createReq.StorageConfigurationId, `ID of the workspace's storage configuration object.`)
	cmd.Flags().StringVar(&createReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", createReq.StorageCustomerManagedKeyId, `The ID of the workspace's storage encryption key configuration object.`)
	cmd.Flags().StringVar(&createReq.WorkspaceName, "workspace-name", createReq.WorkspaceName, `The human-readable name of the workspace.`)

	cmd.Use = "create"
	cmd.Short = `Create a workspace.`
	cmd.Long = `Create a workspace.

  Creates a new workspace using a credential configuration and a storage
  configuration, an optional network configuration (if using a customer-managed
  VPC), an optional managed services key configuration (if using
  customer-managed keys for managed services), and an optional storage key
  configuration (if using customer-managed keys for storage). The key
  configurations used for managed services and storage encryption can be the
  same or different.

  Important: This operation is asynchronous. A response with HTTP status code
  200 means the request has been accepted and is in progress, but does not mean
  that the workspace deployed successfully and is running. The initial workspace
  status is typically PROVISIONING. Use the workspace ID (workspace_id) field in
  the response to identify the new workspace and make repeated GET requests with
  the workspace ID and check its status. The workspace becomes available when
  the status changes to RUNNING.

  You can share one customer-managed VPC with multiple workspaces in a single
  account. It is not required to create a new VPC for each workspace. However,
  you cannot reuse subnets or Security Groups between workspaces. If you plan to
  share one VPC with multiple workspaces, make sure you size your VPC and
  subnets accordingly. Because a Databricks Account API network configuration
  encapsulates this information, you cannot reuse a Databricks Account API
  network configuration across workspaces.

  For information about how to create a new workspace with this API including
  error handling, see [Create a new workspace using the Account API].

  Important: Customer-managed VPCs, PrivateLink, and customer-managed keys are
  supported on a limited set of deployment and subscription types. If you have
  questions about availability, contact your Databricks representative.

  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.

  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

		wait, err := a.Workspaces.Create(ctx, createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *provisioning.Workspace) {
			statusMessage := i.WorkspaceStatusMessage
			spinner <- statusMessage
		}).GetWithTimeout(createTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
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
	*provisioning.DeleteWorkspaceRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq provisioning.DeleteWorkspaceRequest

	cmd.Use = "delete WORKSPACE_ID"
	cmd.Short = `Delete a workspace.`
	cmd.Long = `Delete a workspace.

  Deletes a Databricks workspace, both specified by ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.Workspaces.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*provisioning.GetWorkspaceRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq provisioning.GetWorkspaceRequest

	cmd.Use = "get WORKSPACE_ID"
	cmd.Short = `Get a workspace.`
	cmd.Long = `Get a workspace.

  Gets information including status for a Databricks workspace, specified by ID.
  In the response, the workspace_status field indicates the current status.
  After initial workspace creation (which is asynchronous), make repeated GET
  requests with the workspace ID and check its status. The workspace becomes
  available when the status changes to RUNNING. For information about how to
  create a new workspace with this API **including error handling**, see [Create
  a new workspace using the Account API].

  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.Workspaces.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `List workspaces.`
	cmd.Long = `List workspaces.

  Lists Databricks workspaces for an account.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		response, err := a.Workspaces.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*provisioning.UpdateWorkspaceRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq provisioning.UpdateWorkspaceRequest
	updateReq.CustomerFacingWorkspace = provisioning.Workspace{}
	var updateJson flags.JsonFlag

	var updateSkipWait bool
	var updateTimeout time.Duration

	cmd.Flags().BoolVar(&updateSkipWait, "no-wait", updateSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&updateTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.UpdateMask, "update-mask", updateReq.UpdateMask, `The field mask must be a single string, with multiple fields separated by commas (no spaces).`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.AwsRegion, "aws-region", updateReq.CustomerFacingWorkspace.AwsRegion, ``)
	// TODO: complex arg: azure_workspace_info
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.Cloud, "cloud", updateReq.CustomerFacingWorkspace.Cloud, `The cloud name.`)
	// TODO: complex arg: cloud_resource_container
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.CredentialsId, "credentials-id", updateReq.CustomerFacingWorkspace.CredentialsId, `ID of the workspace's credential configuration object.`)
	// TODO: map via StringToStringVar: custom_tags
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.DeploymentName, "deployment-name", updateReq.CustomerFacingWorkspace.DeploymentName, ``)
	cmd.Flags().Var(&updateReq.CustomerFacingWorkspace.ExpectedWorkspaceStatus, "expected-workspace-status", `A client owned field used to indicate the workspace status that the client expects to be in. Supported values: [
  BANNED,
  CANCELLING,
  FAILED,
  NOT_PROVISIONED,
  PROVISIONING,
  RUNNING,
]`)
	// TODO: complex arg: gcp_managed_network_config
	// TODO: complex arg: gke_config
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.Location, "location", updateReq.CustomerFacingWorkspace.Location, `The Google Cloud region of the workspace data plane in your Google account (for example, us-east4).`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", updateReq.CustomerFacingWorkspace.ManagedServicesCustomerManagedKeyId, `ID of the key configuration for encrypting managed services.`)
	// TODO: complex arg: network
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.NetworkConnectivityConfigId, "network-connectivity-config-id", updateReq.CustomerFacingWorkspace.NetworkConnectivityConfigId, `The object ID of network connectivity config.`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.NetworkId, "network-id", updateReq.CustomerFacingWorkspace.NetworkId, `If this workspace is BYO VPC, then the network_id will be populated.`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.PrivateAccessSettingsId, "private-access-settings-id", updateReq.CustomerFacingWorkspace.PrivateAccessSettingsId, `ID of the workspace's private access settings object.`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.StorageConfigurationId, "storage-configuration-id", updateReq.CustomerFacingWorkspace.StorageConfigurationId, `ID of the workspace's storage configuration object.`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", updateReq.CustomerFacingWorkspace.StorageCustomerManagedKeyId, `ID of the key configuration for encrypting workspace storage.`)
	cmd.Flags().StringVar(&updateReq.CustomerFacingWorkspace.WorkspaceName, "workspace-name", updateReq.CustomerFacingWorkspace.WorkspaceName, `The human-readable name of the workspace.`)

	cmd.Use = "update WORKSPACE_ID"
	cmd.Short = `Update a workspace.`
	cmd.Long = `Update a workspace.

  Updates a workspace.

  Arguments:
    WORKSPACE_ID: A unique integer ID for the workspace`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.CustomerFacingWorkspace)
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
		_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		wait, err := a.Workspaces.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		if updateSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *provisioning.Workspace) {
			statusMessage := i.WorkspaceStatusMessage
			spinner <- statusMessage
		}).GetWithTimeout(updateTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service Workspaces
